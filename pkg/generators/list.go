package generators

import (
	"encoding/json"
	"fmt"
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
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

func (g *ListGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator, _ *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {
	if appSetGenerator == nil {
		return nil, EmptyAppSetGeneratorError
	}

	if appSetGenerator.List == nil {
		return nil, nil
	}

	res := make([]map[string]string, len(appSetGenerator.List.Elements))

	for i, tmpItem := range appSetGenerator.List.Elements {
		params := make(map[string]string, len(tmpItem.Raw))
		var element map[string]interface{}
		err := json.Unmarshal(tmpItem.Raw, &element)
		fmt.Println(element)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshling list element")
		}

		for key, value := range element {
			if key == "values" {
				values, ok := (value).(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("error parsing values map")
				}
				for k, v := range values {
					params[fmt.Sprintf("values.%s", k)] = v.(string)
				}
			} else {
				params[key] = value.(string)
			}
		}

		res[i] = params
	}

	return res, nil
}
