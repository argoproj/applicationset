package generators

import (
	"testing"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMetrixGenerate(t *testing.T) {

	applicationSetInfo := argoprojiov1alpha1.ApplicationSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "set",
		},
		Spec: argoprojiov1alpha1.ApplicationSetSpec{
			Generators: []argoprojiov1alpha1.ApplicationSetGenerator{
				{
					Metrix: []argoprojiov1alpha1.ApplicationSetGenerator{
						{
							Git: &argoprojiov1alpha1.GitGenerator{
								RepoURL:     "RepoURL",
								Revision:    "Revision",
								Directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{Path: "*"}},
							},
							List: &argoprojiov1alpha1.ListGenerator{
								Elements: []argoprojiov1alpha1.ListGeneratorElement{
									{
										Cluster: "Clustre",
										Url:     "Url",
									},
								},
							}},
					},
				},
			},
		},
	}

	var metrixGenerator = NewMertixGenerator()
	got, err := metrixGenerator.GenerateParams(&applicationSetInfo.Spec.Generators[0])

	expected := []map[string]string{}

	assert.NoError(t, err)
	assert.Equal(t, expected, got)

}
