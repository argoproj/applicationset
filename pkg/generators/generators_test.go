package generators

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/argoproj/applicationset/api/v1alpha1"
	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetRelevantGenerators(t *testing.T) {
	requestedGenerator := &v1alpha1.ApplicationSetGenerator{
		List: &v1alpha1.ListGenerator{},
	}
	allGenerators := map[string]Generator{
		"List": NewListGenerator(),
	}
	relevantGenerators := GetRelevantGenerators(requestedGenerator, allGenerators)

	for _, generator := range relevantGenerators {
		if generator == nil {
			t.Fatal(`GetRelevantGenerators produced a nil generator`)
		}
	}

	numRelevantGenerators := len(relevantGenerators)
	if numRelevantGenerators != 1 {
		t.Fatalf(`GetRelevantGenerators produced %d generators instead of the expected 1`, numRelevantGenerators)
	}
}

func TestNoGeneratorNilReferenceError(t *testing.T) {
	generators := []Generator{
		&ClusterGenerator{},
		&DuckTypeGenerator{},
		&GitGenerator{},
		&ListGenerator{},
		&MatrixGenerator{},
		&MergeGenerator{},
		&PullRequestGenerator{},
		&SCMProviderGenerator{},
	}

	for _, generator := range generators {
		testCaseCopy := generator // since tests may run in parallel

		generatorName := reflect.TypeOf(testCaseCopy).Elem().Name()
		t.Run(fmt.Sprintf("%s does not throw a nil reference error when all generator fields are nil", generatorName), func(t *testing.T) {
			t.Parallel()

			params, err := generator.GenerateParams(&v1alpha1.ApplicationSetGenerator{}, &v1alpha1.ApplicationSet{})

			assert.ErrorIs(t, err, EmptyAppSetGeneratorError)
			assert.Nil(t, params)
		})
	}
}

func TestMatchValues(t *testing.T) {
	testCases := []struct {
		name     string
		elements []apiextensionsv1.JSON
		selector *metav1.LabelSelector
		expected []map[string]string
	}{
		{
			name:     "no filter",
			elements: []apiextensionsv1.JSON{{Raw: []byte(`{"cluster": "cluster","url": "url"}`)}},
			selector: &metav1.LabelSelector{},
			expected: []map[string]string{{"cluster": "cluster", "url": "url"}},
		},
		{
			name:     "nil",
			elements: []apiextensionsv1.JSON{{Raw: []byte(`{"cluster": "cluster","url": "url"}`)}},
			selector: nil,
			expected: []map[string]string{{"cluster": "cluster", "url": "url"}},
		},
		{
			name:     "values.foo should be foo but is ignore element",
			elements: []apiextensionsv1.JSON{{Raw: []byte(`{"cluster": "cluster","url": "url","values":{"foo":"bar"}}`)}},
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"values.foo": "foo",
				},
			},
			expected: []map[string]string{},
		},
		{
			name:     "values.foo should be bar",
			elements: []apiextensionsv1.JSON{{Raw: []byte(`{"cluster": "cluster","url": "url","values":{"foo":"bar"}}`)}},
			selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"values.foo": "bar",
				},
			},
			expected: []map[string]string{{"cluster": "cluster", "url": "url", "values.foo": "bar"}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var listGenerator = NewListGenerator()
			var data = map[string]Generator{
				"List": listGenerator,
			}

			results, err := Transform(v1alpha1.ApplicationSetGenerator{
				Selector: testCase.selector,
				List: &v1alpha1.ListGenerator{
					Elements: testCase.elements,
					Template: emptyTemplate(),
				}},
				data,
				emptyTemplate(),
				nil)

			assert.NoError(t, err)
			assert.ElementsMatch(t, testCase.expected, results[0].Params)
		})
	}
}

func emptyTemplate() v1alpha1.ApplicationSetTemplate {
	return v1alpha1.ApplicationSetTemplate{
		Spec: argov1alpha1.ApplicationSpec{
			Project: "project",
		},
	}
}
