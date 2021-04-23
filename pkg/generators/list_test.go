package generators

import (
	"testing"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"

	"github.com/stretchr/testify/assert"
)

func TestGenerateListParams(t *testing.T) {
	testCases := []struct {
		elements []argoprojiov1alpha1.ListGeneratorElement
		expected []map[string]string
	}{
		{
			elements: []argoprojiov1alpha1.ListGeneratorElement{{Cluster: "cluster", Url: "url", Values: map[string]string{}}}, expected: []map[string]string{{
				"cluster": "cluster", "url": "url"},
			},
		},
		{
			elements: []argoprojiov1alpha1.ListGeneratorElement{{Cluster: "cluster", Url: "url", Values: map[string]string{"foo": "bar"}}}, expected: []map[string]string{{
				"cluster": "cluster", "url": "url", "values.foo": "bar",
			}},
		},
	}

	for _, testCase := range testCases {

		var listGenerator = NewListGenerator()

		got, err := listGenerator.GenerateParams(&argoprojiov1alpha1.ApplicationSetGenerator{List: &argoprojiov1alpha1.ListGenerator{
			Elements: testCase.elements,
		}}, nil)

		assert.NoError(t, err)
		assert.ElementsMatch(t, testCase.expected, got)

	}
}
