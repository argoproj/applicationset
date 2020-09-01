package generators

import (
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
)

var _ Generator = (*GitGenerator)(nil)

type GitGenerator struct {
}

func NewGitGenerator() Generator {
	g := &GitGenerator{}
	return g
}

func (g *GitGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {
	return nil, nil
}