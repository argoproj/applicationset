package generators

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
}

func NewClusterGenerator(c client.Client) Generator {
	g := &ClusterGenerator{
		Client: c,
	}
	return g
}

func (g *ClusterGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	return NoRequeueAfter
}

func (g *ClusterGenerator) GenerateParams(
	appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {

	if appSetGenerator == nil {
		return nil, EmptyAppSetGeneratorError
	}

	if appSetGenerator.Clusters == nil {
		return nil, nil
	}

	// List all Clusters:
	clusterSecretList := &corev1.SecretList{}

	selector := metav1.AddLabelToSelector(&appSetGenerator.Clusters.Selector, ArgoCDSecretTypeLabel, ArgoCDSecretTypeCluster)
	secretSelector, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, err
	}

	if err := g.Client.List(context.Background(), clusterSecretList, client.MatchingLabelsSelector{secretSelector}); err != nil {
		return nil, err
	}
	log.Debug("clusters matching labels", "count", len(clusterSecretList.Items))

	// For each matching cluster secret
	res := make([]map[string]string, len(clusterSecretList.Items))
	for i, cluster := range clusterSecretList.Items {
		params := make(map[string]string, len(cluster.ObjectMeta.Annotations)+len(cluster.ObjectMeta.Labels)+2)
		params["name"] = string(cluster.Data["name"])
		params["server"] = string(cluster.Data["server"])
		for key, value := range cluster.ObjectMeta.Annotations {
			params[fmt.Sprintf("metadata.annotations.%s", key)] = value
		}
		for key, value := range cluster.ObjectMeta.Labels {
			params[fmt.Sprintf("metadata.labels.%s", key)] = value
		}
		log.WithField("cluster", cluster.Name).Info("matched cluster secret")

		res[i] = params
	}

	return res, nil
}
