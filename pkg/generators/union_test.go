package generators

import (
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"testing"
)

func getListGenerator(json string) *argoprojiov1alpha1.ListGenerator {
	return &argoprojiov1alpha1.ListGenerator{
		Elements: []apiextensionsv1.JSON{{Raw: []byte(json)}},
	}
}

func TestUnionGenerate(t *testing.T) {

	testCases := []struct {
		name           string
		baseGenerators []argoprojiov1alpha1.ApplicationSetBaseGenerator
		expectedErr    error
		expected       []map[string]string
	}{
		{
			name:           "no generators",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetBaseGenerator{},
			expectedErr:    LessThanTwoGeneratorsInUnion,
		},
		{
			name: "one generator",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetBaseGenerator{
				{
					List: getListGenerator(`{"a": "1_1","b": "same","c": "1_3"}`),
				},
			},
			expectedErr: LessThanTwoGeneratorsInUnion,
		},
		{
			name: "happy flow - generate params",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetBaseGenerator{
				{
					List: getListGenerator(`{"a": "1_1","b": "same","c": "1_3"}`),
				},
				{
					List: getListGenerator(`{"a": "2_1","b": "same"}`),
				},
				{
					List: getListGenerator(`{"a": "3_1","b": "3_2","c": "3_3"}`),
				},
			},
			expected: []map[string]string{
				{"a": "2_1", "b": "same", "c": "1_3"},
				{"a": "3_1", "b": "3_2", "c": "3_3"},
			},
		},
		{
			name: "merge keys absent - do not merge",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetBaseGenerator{
				{
					List: getListGenerator(`{"a": "a"}`),
				},
				{
					List: getListGenerator(`{"a": "a"}`),
				},
			},
			expected: []map[string]string{
				{"a": "a"},
				{"a": "a"},
			},
		},
		{
			name: "merge key present in first set, absent in seconds - do not merge",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetBaseGenerator{
				{
					List: getListGenerator(`{"a": "a"}`),
				},
				{
					List: getListGenerator(`{"b": "b"}`),
				},
			},
			expected: []map[string]string{
				{"a": "a"},
				{"b": "b"},
			},
		},
	}

	for _, testCase := range testCases {
		testCaseCopy := testCase // since tests may run in parallel

		t.Run(testCaseCopy.name, func(t *testing.T) {
			appSet := &argoprojiov1alpha1.ApplicationSet{}

			var UnionGenerator = NewUnionGenerator(
				map[string]Generator{
					"List": &ListGenerator{},
				},
			)

			got, err := UnionGenerator.GenerateParams(&argoprojiov1alpha1.ApplicationSetGenerator{
				Union: &argoprojiov1alpha1.UnionGenerator{
					Generators: testCaseCopy.baseGenerators,
					MergeKeys:  []string{"b"},
					Template:   argoprojiov1alpha1.ApplicationSetTemplate{},
				},
			}, appSet)

			if testCaseCopy.expectedErr != nil {
				assert.EqualError(t, err, testCaseCopy.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCaseCopy.expected, got)
			}

		})

	}
}
