package generators

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

func (a argoCDServiceMock) GetApps(ctx context.Context, repoURL string, revision string) ([]string, error) {
	args := a.Called(ctx, repoURL, revision)

	return args.Get(0).([]string), args.Error(1)
}

func getGitRenderTemplate(name, path string) *argov1alpha1.Application {
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
				Path:           path,
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

func TestGitGenerateApplications(t *testing.T) {

	appSetTemplate := argoprojiov1alpha1.ApplicationSetTemplate{
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
	}

	cases := []struct {
		name		  string
		template      argoprojiov1alpha1.ApplicationSetTemplate
		directories   []argoprojiov1alpha1.GitDirectoryGeneratorItem
		repoApps      []string
		repoError     error
		expected      []argov1alpha1.Application
		expectedError error
	}{
		{
			name: "happy flow - created apps",
			template: appSetTemplate,
			directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{"*"}},
			repoApps: []string{
					"app1",
					"app2",
					"p1/app3",
			},
			repoError: nil,
			expected: []argov1alpha1.Application{
				*getGitRenderTemplate("app1", "app1"),
				*getGitRenderTemplate("app2", "app2"),
			},
			expectedError: nil,
		},
		{
			name: "It filters application according to the paths",
			template: appSetTemplate,
			directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{"p1/*"}, {"p1/*/*"}},
			repoApps: []string{
				"app1",
				"p1/app2",
				"p1/p2/app3",
				"p1/p2/p3/app4",
			},
			repoError: nil,
			expected: []argov1alpha1.Application{
				*getGitRenderTemplate("app2", "p1/app2"),
				*getGitRenderTemplate("app3", "p1/p2/app3"),
			},
			expectedError: nil,
		},
		{
			name: "handles empty response from repo server",
			template: appSetTemplate,
			directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{"*"}},
			repoApps: []string{},
			repoError: nil,
			expected: []argov1alpha1.Application{},
			expectedError:nil,
		},
		{
			name: "handles error from repo server",
			template: appSetTemplate,
			directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{"*"}},
			repoApps: []string{},
			repoError: fmt.Errorf("error"),
			expected: []argov1alpha1.Application{},
			expectedError: fmt.Errorf("error"),
		},
	}

	for _, c := range cases {
		cc := c
		t.Run(cc.name, func(t *testing.T) {
			argoCDServiceMock := argoCDServiceMock{}
			argoCDServiceMock.On("GetApps", mock.Anything, mock.Anything, mock.Anything).Return(c.repoApps, c.repoError)

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
							Directories: c.directories,
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
