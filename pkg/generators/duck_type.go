package generators

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/argoproj/argo-cd/util/settings"
	log "github.com/sirupsen/logrus"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var _ Generator = (*DuckTypeGenerator)(nil)

// DuckTypeGenerator generates Applications for some or all clusters registered with ArgoCD.
type DuckTypeGenerator struct {
	ctx             context.Context
	dynClient       dynamic.Interface
	clientset       kubernetes.Interface
	namespace       string // namespace is the Argo CD namespace
	settingsManager *settings.SettingsManager
}

func NewDuckTypeGenerator(ctx context.Context, dynClient dynamic.Interface, clientset kubernetes.Interface, namespace string) Generator {

	settingsManager := settings.NewSettingsManager(ctx, clientset, namespace)

	g := &DuckTypeGenerator{
		ctx:             ctx,
		dynClient:       dynClient,
		clientset:       clientset,
		namespace:       namespace,
		settingsManager: settingsManager,
	}
	return g
}

func (g *DuckTypeGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {

	// Return a requeue default of 3 minutes, if no override is specified.

	if appSetGenerator.ClusterDecisionResource.RequeueAfterSeconds != nil {
		return time.Duration(*appSetGenerator.ClusterDecisionResource.RequeueAfterSeconds) * time.Second
	}

	return DefaultRequeueAfterSeconds
}

func (g *DuckTypeGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return &appSetGenerator.ClusterDecisionResource.Template
}

func (g *DuckTypeGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator, _ *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {

	if appSetGenerator == nil {
		return nil, EmptyAppSetGeneratorError
	}

	// Not likely to happen
	if appSetGenerator.ClusterDecisionResource == nil {
		return nil, nil
	}

	// ListCluster from Argo CD's util/db package will include the local cluster in the list of clusters
	clustersFromArgoCD, err := utils.ListClusters(g.ctx, g.clientset, g.namespace)
	if err != nil {
		return nil, err
	}

	if clustersFromArgoCD == nil {
		return nil, nil
	}

	// Read the configMapRef
	cm, err := g.clientset.CoreV1().ConfigMaps(g.namespace).Get(g.ctx, appSetGenerator.ClusterDecisionResource.ConfigMapRef, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	// Extract GVK data for the dynamic client to use
	versionIdx := strings.Index(cm.Data["apiVersion"], "/")
	kind := cm.Data["kind"]
	resourceName := appSetGenerator.ClusterDecisionResource.Name

	log.WithField("resourcename.kind.apiVersion", resourceName+"."+kind+"."+
		cm.Data["apiVersion"]).Info("ResourceName.Kind.Group/Version Reference")

	if kind == "" || resourceName == "" || versionIdx < 1 {
		log.Warningf("kind=%v, resourceName=%v, versionIdx=%v", kind, resourceName, versionIdx)
		return nil, errors.New("There is a problem with the apiVersion, kind or resourceName provided")
	}

	// Split up the apiVersion
	group := cm.Data["apiVersion"][0:versionIdx]
	version := cm.Data["apiVersion"][versionIdx+1:]
	log.WithField("kind.group.version", kind+"."+group+"/"+version).Debug("decoded Ref")

	duckGVR := schema.GroupVersionResource{Group: group, Version: version, Resource: kind}
	duckResource, err := g.dynClient.Resource(duckGVR).Namespace(g.namespace).Get(g.ctx, resourceName, metav1.GetOptions{})

	if err != nil {
		log.WithField("GVK", duckGVR).Warningf("resource was not found with name=%v", resourceName)
		return nil, err
	}

	if duckResource == nil || duckResource.Object["status"] == nil {
		log.Warning("duck type status missing")
		return nil, nil
	}

	log.WithField("duckResourceName", duckResource.GetName()).Debug("found duck status on resource")

	// Override the duck type in the status of the resource
	statusListKey := "clusters"

	matchKey := cm.Data["matchKey"]

	if cm.Data["statusListKey"] != "" {
		statusListKey = cm.Data["statusListKey"]
	}
	if matchKey == "" {
		log.WithField("matchKey", matchKey).Warning("matchKey not found in " + cm.Name)
		return nil, nil

	}

	res := []map[string]string{}

	clusterDecisions := duckResource.Object["status"].(map[string]interface{})[statusListKey]

	if clusterDecisions != nil {
		for _, cluster := range clusterDecisions.([]interface{}) {

			// generated instance of cluster params
			params := map[string]string{}

			matchValue := cluster.(map[string]interface{})[matchKey]
			if matchValue == nil || matchValue.(string) == "" {
				log.Warningf("matchKey=%v not found in \"%v\" list: %v\n", matchKey, statusListKey, cluster.(map[string]interface{}))
				continue
			}

			strMatchValue := matchValue.(string)
			log.WithField(matchKey, strMatchValue).Debug("validate against ArgoCD")

			found := false

			for _, argoCluster := range clustersFromArgoCD.Items {
				if argoCluster.Name == strMatchValue {

					log.WithField(matchKey, argoCluster.Name).Info("matched cluster in ArgoCD")
					params["name"] = argoCluster.Name
					params["server"] = argoCluster.Server

					found = true
				}

			}

			if !found {
				log.WithField(matchKey, strMatchValue).Warning("unmatched cluster in ArgoCD")
				continue
			}

			for key, value := range cluster.(map[string]interface{}) {
				params[key] = value.(string)
			}

			for key, value := range appSetGenerator.ClusterDecisionResource.Values {
				params[fmt.Sprintf("values.%s", key)] = value
			}

			res = append(res, params)
		}
	} else {
		return nil, errors.New("duck type status." + statusListKey + " missing")
	}

	return res, nil
}
