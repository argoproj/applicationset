package generators

import (
	"testing"
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMetrixGenerate(t *testing.T) {

	gitGeneratorSpec := argoprojiov1alpha1.ApplicationSetGenerator{
		Git: &argoprojiov1alpha1.GitGenerator{
			RepoURL:     "RepoURL",
			Revision:    "Revision",
			Directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{Path: "*"}},
		},
	}

	applicationSetInfo := argoprojiov1alpha1.ApplicationSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "set",
		},
		Spec: argoprojiov1alpha1.ApplicationSetSpec{
			Generators: []argoprojiov1alpha1.ApplicationSetGenerator{
				{
					Metrix: &argoprojiov1alpha1.MetrixGenerator{
						Generators: []argoprojiov1alpha1.ApplicationSetGenerator{
							gitGeneratorSpec,
							{
								List: &argoprojiov1alpha1.ListGenerator{
									Elements: []argoprojiov1alpha1.ListGeneratorElement{
										{
											Cluster: "Cluster",
											Url:     "Url",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	mock := &generatorMock{}

	mock.On("GenerateParams", &gitGeneratorSpec).Return([]map[string]string{
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

	var metrixGenerator = NewMertixGenerator(
		map[string]Generator{
			"Git":  mock,
			"List": &ListGenerator{},
		},
	)
	got, err := metrixGenerator.GenerateParams(&applicationSetInfo.Spec.Generators[0])

	expected := []map[string]string{
		{"path": "app1", "path.basename": "app1", "cluster": "Cluster", "url": "Url"},
		{"path": "app2", "path.basename": "app2", "cluster": "Cluster", "url": "Url"},
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, got)

}

type generatorMock struct {
	mock.Mock
}

func (g *generatorMock) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	args := g.Called(appSetGenerator)

	return args.Get(0).(*argoprojiov1alpha1.ApplicationSetTemplate)
}

func (g *generatorMock) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {
	args := g.Called(appSetGenerator)

	return args.Get(0).([]map[string]string), args.Error(1)
}

func (g *generatorMock) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	args := g.Called(appSetGenerator)

	return args.Get(0).(time.Duration)

}
