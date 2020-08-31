package git

import (
	"context"
	"fmt"
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/reposerver/apiclient"
	"github.com/argoproj/argo-cd/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type clientSet struct {
	RepoServerServiceClient apiclient.RepoServerServiceClient
}

func (c *clientSet) NewRepoServerClient() (util.Closer, apiclient.RepoServerServiceClient, error) {
	return util.NewCloser(func() error { return nil }), c.RepoServerServiceClient, nil
}

type argoCDServiceMock struct {
	mock.Mock
}

func (a argoCDServiceMock) GetApps(ctx context.Context, repoURL string, revision string, path string) ([]string, error) {
	args := a.Called(ctx, repoURL, revision, path)

	return args.Get(0).([]string), args.Error(1)
}

func getRenderTemplate(name string) *argov1alpha1.Application {
	return &argov1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "namespace",
			Finalizers: []string{
				"resources-finalizer.argocd.argoproj.io",
			},
		},
		Spec: argov1alpha1.ApplicationSpec{
			Source: argov1alpha1.ApplicationSource{
				RepoURL:        "RepoURL",
				Path:           name,
				TargetRevision: "HEAD",
			},
			Destination: argov1alpha1.ApplicationDestination{
				Server:    "server",
				Namespace: "destinationNamespace",
			},
			Project: "project",
		},
	}
}

func TestGenerateApplications(t *testing.T) {
	cases := []struct {
		name		  string
		template      argoprojiov1alpha1.ApplicationSetTemplate
		Directories   []argoprojiov1alpha1.GitDirectoryGeneratorItem
		repoApps      []string
		repoError     error
		expected      []argov1alpha1.Application
		expectedError error
	}{
		{
			"happy flow",
			argoprojiov1alpha1.ApplicationSetTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "{{path.basename}}",
					Namespace: "namespace",
				},
				Spec: argov1alpha1.ApplicationSpec{
					Source: argov1alpha1.ApplicationSource{
						RepoURL:        "RepoURL",
						Path:           "{{path}}",
						TargetRevision: "HEAD",
					},
					Destination: argov1alpha1.ApplicationDestination{
						Server:    "server",
						Namespace: "destinationNamespace",
					},
					Project: "project",
				},
			},
			[]argoprojiov1alpha1.GitDirectoryGeneratorItem{{"path"}},
			[]string{
					"app1",
					"app2",
			},
			nil,
			[]argov1alpha1.Application{
				*getRenderTemplate("app1"),
				*getRenderTemplate("app2"),
			},
			nil,
		},
		{
			"handles empty response from repo server",
			argoprojiov1alpha1.ApplicationSetTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "{{path.basename}}",
					Namespace: "namespace",
				},
				Spec: argov1alpha1.ApplicationSpec{
					Source: argov1alpha1.ApplicationSource{
						RepoURL:        "RepoURL",
						Path:           "{{path}}",
						TargetRevision: "HEAD",
					},
					Destination: argov1alpha1.ApplicationDestination{
						Server:    "server",
						Namespace: "destinationNamespace",
					},
					Project: "project",
				},
			},
			[]argoprojiov1alpha1.GitDirectoryGeneratorItem{{"path"}},
			[]string{},
			nil,
			[]argov1alpha1.Application{},
			nil,
		},
		{
			"handles error from repo server",
			argoprojiov1alpha1.ApplicationSetTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "{{path.basename}}",
					Namespace: "namespace",
				},
				Spec: argov1alpha1.ApplicationSpec{
					Source: argov1alpha1.ApplicationSource{
						RepoURL:        "RepoURL",
						Path:           "{{path}}",
						TargetRevision: "HEAD",
					},
					Destination: argov1alpha1.ApplicationDestination{
						Server:    "server",
						Namespace: "destinationNamespace",
					},
					Project: "project",
				},
			},
			[]argoprojiov1alpha1.GitDirectoryGeneratorItem{{"path"}},
			[]string{},
			fmt.Errorf("error"),
			[]argov1alpha1.Application{},
			nil,
		},
	}

	for _, c := range cases {
		cc := c
		t.Run(cc.name, func(t *testing.T) {
			argoCDServiceMock := argoCDServiceMock{}
			argoCDServiceMock.On("GetApps", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(c.repoApps, c.repoError)

			var gitGenerator = NewGitGenerator(argoCDServiceMock)
			applicationSetInfo := argoprojiov1alpha1.ApplicationSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "set",
				},
				Spec: argoprojiov1alpha1.ApplicationSetSpec{
					Generators: []argoprojiov1alpha1.ApplicationSetGenerator{{
						Git: &argoprojiov1alpha1.GitGenerator{
							RepoURL:     "RepoURL",
							Revision:    "Revision",
							Directories: c.Directories,
						},
					}},
					Template: c.template,
				},
			}

			got, err := gitGenerator.GenerateApplications(&applicationSetInfo.Spec.Generators[0], &applicationSetInfo)

			if c.expectedError != nil {
				assert.EqualError(t, err, c.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expected, got)
			}

			argoCDServiceMock.AssertExpectations(t)
		})
	}

}
