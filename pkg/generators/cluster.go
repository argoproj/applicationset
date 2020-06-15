package generators

import (
	"fmt"
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
)

var _ Generator = (*ClusterGenerator)(nil)

// ClusterGenerator generates Applications for some or all clusters registered with ArgoCD.
type ClusterGenerator struct {
}

func NewClusterGenerator() Generator {
	// TODO: pass client or informer for access to cluster secrets
	g := &ClusterGenerator{}
	return g
}

func (g *ClusterGenerator) GenerateApplications(appSet *argoprojiov1alpha1.ApplicationSet) ([]argov1alpha1.Application, error) {
	if appSet == nil {
		return nil, fmt.Errorf("ApplicationSet is empty")
	}

	return nil, nil
}
