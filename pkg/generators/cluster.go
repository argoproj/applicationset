package generators

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/utils"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
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

	var tmplApplication argov1alpha1.Application
	tmplApplication.Namespace = appSet.Spec.Template.Namespace
	tmplApplication.Name = appSet.Spec.Template.Name
	tmplApplication.Spec = appSet.Spec.Template.Spec

	var resultingApplications []argov1alpha1.Application

	for _, cluster := range clusterSecretList.Items {
		params := make(map[string]string)
		params["name"] = cluster.Name
		params["server"] = string(cluster.Data["server"])
		for key, value := range cluster.ObjectMeta.Labels {
			params[fmt.Sprintf("metadata.labels.%s", key)] = value
		}
		log.WithField("cluster", cluster.Name).Info("matched cluster secret")
		tmpApplication, err := utils.RenderTemplateParams(&tmplApplication, params)
		if err != nil {
			log.WithField("cluster", cluster.Name).Error("Error during rendering template params")
			continue
		}
		resultingApplications = append(resultingApplications, *tmpApplication)
	}

	return resultingApplications, nil
}
