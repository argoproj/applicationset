package generators

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
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

func (g *ClusterGenerator) GenerateApplications(
	appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator,
	appSet *argoprojiov1alpha1.ApplicationSet) ([]argov1alpha1.Application, error) {

	if appSetGenerator == nil {
		return nil, fmt.Errorf("ApplicationSetGenerator is empty")
	}
	if appSet == nil {
		return nil, fmt.Errorf("ApplicationSet is empty")
	}

	// List all Clusters:
	clusterSecretList := &corev1.SecretList{}
	secretLabels := map[string]string{
		ArgoCDSecretTypeLabel: ArgoCDSecretTypeCluster,
	}
	for k, v := range appSetGenerator.Clusters.Selector.MatchLabels {
		secretLabels[k] = v
	}
	if err := g.Client.List(context.Background(), clusterSecretList, client.MatchingLabels(secretLabels)); err != nil {
		return nil, err
	}
	log.Debug("clusters matching labels", "count", len(clusterSecretList.Items))

	for _, cluster := range clusterSecretList.Items {
		log.WithField("cluster", cluster.Name).Info("matched cluster secret")
	}

	return nil, nil
}
