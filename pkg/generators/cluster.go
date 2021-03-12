package generators

import (
	"context"
	"fmt"
	"time"

	"github.com/argoproj/argo-cd/util/db"
	"github.com/argoproj/argo-cd/util/settings"
	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
)

const (
	ArgoCDSecretTypeLabel   = "argocd.argoproj.io/secret-type"
	ArgoCDSecretTypeCluster = "cluster"
)

var _ Generator = (*ClusterGenerator)(nil)

// ClusterGenerator generates Applications for some or all clusters registered with ArgoCD.
type ClusterGenerator struct {
	client.Client
	ctx       context.Context
	clientset kubernetes.Interface
	// namespace is the Argo CD namespace
	namespace       string
	settingsManager *settings.SettingsManager
	argoDB          db.ArgoDB
}

func NewClusterGenerator(c client.Client, ctx context.Context, clientset kubernetes.Interface, namespace string) Generator {

	settingsManager := settings.NewSettingsManager(ctx, clientset, namespace)

	g := &ClusterGenerator{
		Client:          c,
		ctx:             ctx,
		clientset:       clientset,
		namespace:       namespace,
		settingsManager: settingsManager,
		argoDB:          db.NewDB(namespace, settingsManager, clientset),
	}
	return g
}

func (g *ClusterGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	return NoRequeueAfter
}

func (g *ClusterGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return &appSetGenerator.Clusters.Template
}

func (g *ClusterGenerator) GenerateParams(
	appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {

	if appSetGenerator == nil {
		return nil, EmptyAppSetGeneratorError
	}

	if appSetGenerator.Clusters == nil {
		return nil, nil
	}

	// Do not include the local cluster in the cluster parameters IF there is a non-empty selector
	// - Since local clusters do not have secrets, they do not have labels to match against
	ignoreLocalClusters := len(appSetGenerator.Clusters.Selector.MatchExpressions) > 0 || len(appSetGenerator.Clusters.Selector.MatchLabels) > 0

	// ListCluster from Argo CD's util/db package will include the local cluster in the list of clusters
	clustersFromArgoCD, err := g.argoDB.ListClusters(g.ctx)
	if err != nil {
		return nil, err
	}

	if clustersFromArgoCD == nil {
		return nil, nil
	}

	clusterSecrets, err := g.getSecretsByClusterName(appSetGenerator)
	if err != nil {
		return nil, err
	}

	res := []map[string]string{}

	secretsFound := []corev1.Secret{}

	for _, cluster := range clustersFromArgoCD.Items {

		// If there is a secret for this cluster, then it's a non-local cluster, so it will be
		// handled by the next step.
		if secretForCluster, exists := clusterSecrets[cluster.Name]; exists {
			secretsFound = append(secretsFound, secretForCluster)

		} else if !ignoreLocalClusters {
			// If there is no secret for the cluster, it's the local cluster, so handle it here.
			params := map[string]string{}
			params["name"] = cluster.Name
			params["server"] = cluster.Server

			for key, value := range appSetGenerator.Clusters.Values {
				params[fmt.Sprintf("values.%s", key)] = value
			}

			log.WithField("cluster", "local cluster").Info("matched local cluster")

			res = append(res, params)
		}
	}

	// For each matching cluster secret (non-local clusters only)
	for _, cluster := range secretsFound {
		params := map[string]string{}
		params["name"] = string(cluster.Data["name"])
		params["server"] = string(cluster.Data["server"])
		for key, value := range cluster.ObjectMeta.Annotations {
			params[fmt.Sprintf("metadata.annotations.%s", key)] = value
		}
		for key, value := range cluster.ObjectMeta.Labels {
			params[fmt.Sprintf("metadata.labels.%s", key)] = value
		}
		for key, value := range appSetGenerator.Clusters.Values {
			params[fmt.Sprintf("values.%s", key)] = value
		}
		log.WithField("cluster", cluster.Name).Info("matched cluster secret")

		res = append(res, params)
	}

	return res, nil
}

func (g *ClusterGenerator) getSecretsByClusterName(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) (map[string]corev1.Secret, error) {
	// List all Clusters:
	clusterSecretList := &corev1.SecretList{}

	selector := metav1.AddLabelToSelector(&appSetGenerator.Clusters.Selector, ArgoCDSecretTypeLabel, ArgoCDSecretTypeCluster)
	secretSelector, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, err
	}

	if err := g.Client.List(context.Background(), clusterSecretList, client.MatchingLabelsSelector{Selector: secretSelector}); err != nil {
		return nil, err
	}
	log.Debug("clusters matching labels", "count", len(clusterSecretList.Items))

	res := map[string]corev1.Secret{}

	for _, cluster := range clusterSecretList.Items {
		clusterName := string(cluster.Data["name"])

		res[clusterName] = cluster
	}

	return res, nil

}
