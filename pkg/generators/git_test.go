package generators

import (
	"context"
	"fmt"
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj/argo-cd/reposerver/apiclient"
	"github.com/argoproj/gitops-engine/pkg/utils/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type clientSet struct {
	RepoServerServiceClient apiclient.RepoServerServiceClient
}

func (c *clientSet) NewRepoServerClient() (io.Closer, apiclient.RepoServerServiceClient, error) {
	return io.NewCloser(func() error { return nil }), c.RepoServerServiceClient, nil
}

type argoCDServiceMock struct {
	mock.Mock
}

func (a argoCDServiceMock) GetApps(ctx context.Context, repoURL string, revision string) ([]string, error) {
	args := a.Called(ctx, repoURL, revision)

	return args.Get(0).([]string), args.Error(1)
}

func TestGitGenerateParams(t *testing.T) {

	cases := []struct {
		name          string
		directories   []argoprojiov1alpha1.GitDirectoryGeneratorItem
		repoApps      []string
		repoError     error
		expected      []map[string]string
		expectedError error
	}{
		{
			name:        "happy flow - created apps",
			directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{"*"}},
			repoApps: []string{
				"app1",
				"app2",
				"p1/app3",
			},
			repoError: nil,
			expected: []map[string]string{
				{"path": "app1", "path.basename": "app1"},
				{"path": "app2", "path.basename": "app2"},
			},
			expectedError: nil,
		},
		{
			name:        "It filters application according to the paths",
			directories: []argoprojiov1alpha1.GitDirectoryGeneratorItem{{"p1/*"}, {"p1/*/*"}},
			repoApps: []string{
				"app1",
				"p1/app2",
				"p1/p2/app3",
				"p1/p2/p3/app4",
			},
			repoError: nil,
			expected: []map[string]string{
				{"path": "p1/app2", "path.basename": "app2"},
				{"path": "p1/p2/app3", "path.basename": "app3"},
			},
			expectedError: nil,
		},
		{
			name:          "handles empty response from repo server",
			directories:   []argoprojiov1alpha1.GitDirectoryGeneratorItem{{"*"}},
			repoApps:      []string{},
			repoError:     nil,
			expected:      []map[string]string{},
			expectedError: nil,
		},
		{
			name:          "handles error from repo server",
			directories:   []argoprojiov1alpha1.GitDirectoryGeneratorItem{{"*"}},
			repoApps:      []string{},
			repoError:     fmt.Errorf("error"),
			expected:      []map[string]string{},
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
				},
			}

			got, err := gitGenerator.GenerateParams(&applicationSetInfo.Spec.Generators[0])

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
