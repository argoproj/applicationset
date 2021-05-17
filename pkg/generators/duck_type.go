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

	if appSetGenerator.DuckType.RequeueAfterSeconds != nil {
		return time.Duration(*appSetGenerator.DuckType.RequeueAfterSeconds) * time.Second
	}

	return DefaultRequeueAfterSeconds
}

func (g *DuckTypeGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return &appSetGenerator.DuckType.Template
}

func (g *DuckTypeGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator, _ *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {

	if appSetGenerator == nil {
		return nil, EmptyAppSetGeneratorError
	}

	// Not likely to happen
	if appSetGenerator.DuckType == nil {
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

	// Read the duck resource, so the status can be examined
	versionIdx := strings.Index(appSetGenerator.DuckType.ApiVersion, "/")
	kind := appSetGenerator.DuckType.Kind
	resourceName := appSetGenerator.DuckType.Name

	log.WithField("resourcename.kind.apiVersion", resourceName+"."+kind+"."+
		appSetGenerator.DuckType.ApiVersion).Info("ResourcenameGroupVersionKind Reference")

	if kind == "" || resourceName == "" || versionIdx < 1 {
		return nil, errors.New("Invalid resource reference")
	}

	// Split up the apiVersion
	group := appSetGenerator.DuckType.ApiVersion[0:versionIdx]
	version := appSetGenerator.DuckType.ApiVersion[versionIdx+1:]
	log.WithField("resourceName.kind.group.version", kind+"."+group+"/"+version).Debug("decoded Ref")

	duckGVR := schema.GroupVersionResource{Group: group, Version: version, Resource: kind}
	duckResource, err := g.dynClient.Resource(duckGVR).Namespace(g.namespace).Get(g.ctx, resourceName, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	if duckResource == nil || duckResource.Object["status"] == nil {
		log.Warning("duck type status missing")
		return nil, nil
	}

	log.WithField("duckResourceName", duckResource.GetName()).Debug("found duck resource")

	// We will use a map to detect if a cluster is in the Status.Decisions array from the Duck Type,
	// this gives us a X+n cost, where n is the number of clusters and X is the number of decisions
	clusterDecisionMap := map[string]bool{}

	if duckResource.Object["status"].(map[string]interface{})["decisions"] != nil {
		for _, cluster := range duckResource.Object["status"].(map[string]interface{})["decisions"].([]interface{}) {

			clusterName := cluster.(map[string]interface{})["clusterName"].(string)
			log.WithField("cluster", clusterName).Debug("found cluster")

			clusterDecisionMap[clusterName] = true
		}
	} else {
		log.Warning("duck type status.decisions missing")
	}

	res := []map[string]string{}

	for _, cluster := range clustersFromArgoCD.Items {
		if clusterDecisionMap[cluster.Name] {

			params := map[string]string{}
			params["name"] = cluster.Name
			params["server"] = cluster.Server

			for key, value := range appSetGenerator.DuckType.Values {
				params[fmt.Sprintf("values.%s", key)] = value
			}

			log.WithField("cluster", cluster.Name).Info("matched cluster")

			res = append(res, params)
		}
	}

	if len(clusterDecisionMap) < len(res) {
		log.Infof("Decisions list of %v, only matched %v ArgoCD clusters", len(clusterDecisionMap), len(res))
	}

	return res, nil
}
