package generators

import (
	"fmt"
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
)

var _ Generator = (*GitGenerator)(nil)

type GitGenerator struct {
}

func NewGitGenerator() Generator {
	g := &GitGenerator{}
	return g
}

func (g *GitGenerator) GenerateApplications(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator, appSet *argoprojiov1alpha1.ApplicationSet) ([]argov1alpha1.Application, error) {
	if appSet == nil {
		return nil, fmt.Errorf("ApplicationSet is empty")
	}

	return nil, nil
}
