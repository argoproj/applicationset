package generators

import (
	"fmt"
	"time"

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

func (g *ListGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	return NoRequeueAfter
}

func (g *ListGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return &appSetGenerator.List.Template
}

func (g *ListGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {
	if appSetGenerator == nil {
		return nil, EmptyAppSetGeneratorError
	}

	if appSetGenerator.List == nil {
		return nil, nil
	}

	res := make([]map[string]string, len(appSetGenerator.List.Elements))

	for i, tmpItem := range appSetGenerator.List.Elements {
		params := make(map[string]string, 2+len(tmpItem.Values))
		params[utils.ClusterListGeneratorKeyName] = tmpItem.Cluster
		params[utils.UrlGeneratorKeyName] = tmpItem.Url
		for key, value := range tmpItem.Values {
			params[fmt.Sprintf("values.%s", key)] = value
		}
		res[i] = params
	}

	return res, nil
}
