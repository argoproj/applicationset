package generators

import (
	"testing"
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestMatrixGenerate(t *testing.T) {

	gitGenerator := &argoprojiov1alpha1.GitGenerator{
		RepoURL:     "RepoURL",
		Revision:    "Revision",
		Directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{Path: "*"}},
	}

	listGenerator := &argoprojiov1alpha1.ListGenerator{
		Elements: []apiextensionsv1.JSON{{Raw: []byte(`{"cluster": "Cluster","url": "Url"}`)}},
	}

	testCases := []struct {
		name           string
		baseGenerators []argoprojiov1alpha1.ApplicationSetNestedGenerator
		expectedErr    error
		expected       []map[string]string
	}{
		{
			name: "happy flow - generate params",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetNestedGenerator{
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						Git: gitGenerator,
					},
				},
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						List: listGenerator,
					},
				},
			},
			expected: []map[string]string{
				{"path": "app1", "path.basename": "app1", "cluster": "Cluster", "url": "Url"},
				{"path": "app2", "path.basename": "app2", "cluster": "Cluster", "url": "Url"},
			},
		},
		{
			name: "happy flow - generate params from two lists",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetNestedGenerator{
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						List: &argoprojiov1alpha1.ListGenerator{
							Elements: []apiextensionsv1.JSON{
								{Raw: []byte(`{"a": "1"}`)},
								{Raw: []byte(`{"a": "2"}`)},
							},
						},
					},
				},
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						List: &argoprojiov1alpha1.ListGenerator{
							Elements: []apiextensionsv1.JSON{
								{Raw: []byte(`{"b": "1"}`)},
								{Raw: []byte(`{"b": "2"}`)},
							},
						},
					},
				},
			},
			expected: []map[string]string{
				{"a": "1", "b": "1"},
				{"a": "1", "b": "2"},
				{"a": "2", "b": "1"},
				{"a": "2", "b": "2"},
			},
		},
		{
			name: "returns error if there is less than two base generators",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetNestedGenerator{
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						Git: gitGenerator,
					},
				},
			},
			expectedErr: LessThanTwoGenerators,
		},
		{
			name: "returns error if there is more than two base generators",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetNestedGenerator{
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						List: listGenerator,
					},
				},
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						List: listGenerator,
					},
				},
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						List: listGenerator,
					},
				},
			},
			expectedErr: MoreThanTwoGenerators,
		},
		{
			name: "returns error if there is more than one inner generator in the first base generator",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetNestedGenerator{
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						Git:  gitGenerator,
						List: listGenerator,
					},
				},
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						Git: gitGenerator,
					},
				},
			},
			expectedErr: MoreThenOneInnerGenerators,
		},
		{
			name: "returns error if there is more than one inner generator in the second base generator",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetNestedGenerator{
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						List: listGenerator,
					},
				},
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						Git:  gitGenerator,
						List: listGenerator,
					},
				},
			},
			expectedErr: MoreThenOneInnerGenerators,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mock := &generatorMock{}
			appSet := &argoprojiov1alpha1.ApplicationSet{}

			for _, g := range testCase.baseGenerators {

				gitGeneratorSpec := argoprojiov1alpha1.ApplicationSetGenerator{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						Git:  g.Git,
						List: g.List,
					},
				}
				mock.On("GenerateParams", &gitGeneratorSpec, appSet).Return([]map[string]string{
					{
						"path":          "app1",
						"path.basename": "app1",
					},
					{
						"path":          "app2",
						"path.basename": "app2",
					},
				}, nil)

				mock.On("GetTemplate", &gitGeneratorSpec).
					Return(&argoprojiov1alpha1.ApplicationSetTemplate{})
			}

			var matrixGenerator = NewMatrixGenerator(
				map[string]Generator{
					"Git":  mock,
					"List": &ListGenerator{},
				},
			)

			got, err := matrixGenerator.GenerateParams(&argoprojiov1alpha1.ApplicationSetGenerator{
				Matrix: &argoprojiov1alpha1.MatrixGenerator{
					Generators: testCase.baseGenerators,
					Template:   argoprojiov1alpha1.ApplicationSetTemplate{},
				},
			}, appSet)

			if testCase.expectedErr != nil {
				assert.EqualError(t, err, testCase.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, got)
			}

		})

	}
}

func TestMatrixGetRequeueAfter(t *testing.T) {

	gitGenerator := &argoprojiov1alpha1.GitGenerator{
		RepoURL:     "RepoURL",
		Revision:    "Revision",
		Directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{Path: "*"}},
	}

	listGenerator := &argoprojiov1alpha1.ListGenerator{
		Elements: []apiextensionsv1.JSON{{Raw: []byte(`{"cluster": "Cluster","url": "Url"}`)}},
	}

	testCases := []struct {
		name               string
		baseGenerators     []argoprojiov1alpha1.ApplicationSetNestedGenerator
		gitGetRequeueAfter time.Duration
		expected           time.Duration
	}{
		{
			name: "return NoRequeueAfter if all the inner baseGenerators returns it",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetNestedGenerator{
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						Git: gitGenerator,
					},
				},
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						List: listGenerator,
					},
				},
			},
			gitGetRequeueAfter: NoRequeueAfter,
			expected:           NoRequeueAfter,
		},
		{
			name: "returns the minimal time",
			baseGenerators: []argoprojiov1alpha1.ApplicationSetNestedGenerator{
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						Git: gitGenerator,
					},
				},
				{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						List: listGenerator,
					},
				},
			},
			gitGetRequeueAfter: time.Duration(1),
			expected:           time.Duration(1),
		},
	}

	for _, c := range testCases {
		cc := c

		t.Run(cc.name, func(t *testing.T) {
			mock := &generatorMock{}

			for _, g := range cc.baseGenerators {
				gitGeneratorSpec := argoprojiov1alpha1.ApplicationSetGenerator{
					ApplicationSetTerminalGenerator: &argoprojiov1alpha1.ApplicationSetTerminalGenerator{
						Git:  g.Git,
						List: g.List,
					},
				}
				mock.On("GetRequeueAfter", &gitGeneratorSpec).Return(cc.gitGetRequeueAfter, nil)
			}

			var matrixGenerator = NewMatrixGenerator(
				map[string]Generator{
					"Git":  mock,
					"List": &ListGenerator{},
				},
			)

			got := matrixGenerator.GetRequeueAfter(&argoprojiov1alpha1.ApplicationSetGenerator{
				Matrix: &argoprojiov1alpha1.MatrixGenerator{
					Generators: cc.baseGenerators,
					Template:   argoprojiov1alpha1.ApplicationSetTemplate{},
				},
			})

			assert.Equal(t, cc.expected, got)

		})

	}
}

type generatorMock struct {
	mock.Mock
}

func (g *generatorMock) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	args := g.Called(appSetGenerator)

	return args.Get(0).(*argoprojiov1alpha1.ApplicationSetTemplate)
}

func (g *generatorMock) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator, appSet *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {
	args := g.Called(appSetGenerator, appSet)

	return args.Get(0).([]map[string]string), args.Error(1)
}

func (g *generatorMock) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	args := g.Called(appSetGenerator)

	return args.Get(0).(time.Duration)

}
