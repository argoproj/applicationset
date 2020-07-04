package generators

import (
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj/argo-cd/reposerver/apiclient"
	"github.com/argoproj/argo-cd/reposerver/apiclient/mocks"
	"github.com/argoproj/argo-cd/util"
	"github.com/stretchr/testify/mock"
	"testing"
)

type Clientset struct {
	RepoServerServiceClient apiclient.RepoServerServiceClient
}

func (c *Clientset) NewRepoServerClient() (util.Closer, apiclient.RepoServerServiceClient, error) {
	return util.NewCloser(func() error{ return nil}), c.RepoServerServiceClient, nil
}

func TestGenerateApplications(t *testing.T) {
	cases := []struct {
		template argoprojiov1alpha1.ApplicationSetTemplate
		Directories []argoprojiov1alpha1.GitDirectoryGeneratorItem
		repoApps *apiclient.AppList
	}{
		{
			argoprojiov1alpha1.ApplicationSetTemplate{},
			[]argoprojiov1alpha1.GitDirectoryGeneratorItem{{"path"}},
			&apiclient.AppList{},
		},
	}

	for _, c := range cases {
		mockRepoServiceClient := mocks.RepoServerServiceClient{}
		mockRepoServiceClient.On("ListApps", mock.Anything, mock.Anything).Return(c.repoApps, nil)
		mockRepoClient := &Clientset{RepoServerServiceClient: &mockRepoServiceClient}

		var gitGenerator = NewGitGenerator(mockRepoClient)
		applicationSetInfo := argoprojiov1alpha1.ApplicationSet{
			Spec: argoprojiov1alpha1.ApplicationSetSpec{
				Generators: []argoprojiov1alpha1.ApplicationSetGenerator{{
					Git: &argoprojiov1alpha1.GitGenerator{
						RepoURL:     "RepoURL",
						Revision:    "Revision",
						Directories: c.Directories,
					},
				},},
				Template: c.template,
			},
		}

		_, _ = gitGenerator.GenerateApplications(&applicationSetInfo.Spec.Generators[0], &applicationSetInfo )
	}


}