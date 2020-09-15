package generators

import (
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/utils"
)

var _ Generator = (*ListGenerator)(nil)

type ListGenerator struct {

}

func NewListGenerator() Generator {
	g := &ListGenerator{}
	return g
}

func (g *ListGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {
	if appSetGenerator == nil {
		return nil, EmptyAppSetGeneratorError
	}

	if appSetGenerator.List == nil {
		return nil, nil
	}

	res := make([]map[string]string, len(appSetGenerator.List.Elements))

	params := make(map[string]string, 2)
	for _, tmpItem := range appSetGenerator.List.Elements {
		params[utils.ClusterListGeneratorKeyName] = tmpItem.Cluster
		params[utils.UrlGeneratorKeyName] = tmpItem.Url
		res = append(res, params)
	}

	return res, nil
}
