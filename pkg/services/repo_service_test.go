package services

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/reposerver/apiclient"
	"github.com/argoproj/argo-cd/util/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type ArgocdRepositoryMock struct {
	mock *mock.Mock
}

func (a ArgocdRepositoryMock) GetRepository(ctx context.Context, url string) (*v1alpha1.Repository, error) {
	args := a.mock.Called(ctx, url)

	return args.Get(0).(*v1alpha1.Repository), args.Error(1)

}

type closer struct {
	// mock *mock.Mock
}

func (c closer) Close() error {
	return nil
}

type repoClientsetMock struct {
	mock *mock.Mock
}

func (r repoClientsetMock) NewRepoServerClient() (io.Closer, apiclient.RepoServerServiceClient, error) {
	args := r.mock.Called()

	return closer{}, args.Get(0).(apiclient.RepoServerServiceClient), args.Error(1)
}

func TestGetDirectories(t *testing.T) {

	// Hardcode a specific revision to changes to argocd-example-apps from regressing this test:
	//   Author: Alexander Matyushentsev <Alexander_Matyushentsev@intuit.com>
	//   Date:   Sun Jan 31 09:54:53 2021 -0800
	//   chore: downgrade kustomize guestbook image tag (#73)
	exampleRepoRevision := "08f72e2a309beab929d9fd14626071b1a61a47f9"

	for _, c := range []struct {
		name          string
		repoURL       string
		revision      string
		repoRes       *v1alpha1.Repository
		repoErr       error
		expected      []string
		expectedError error
	}{
		{
			name:     "All child folders should be returned",
			repoURL:  "https://github.com/argoproj/argocd-example-apps/",
			revision: exampleRepoRevision,
			repoRes: &v1alpha1.Repository{
				Repo: "https://github.com/argoproj/argocd-example-apps/",
			},
			repoErr: nil,
			expected: []string{"apps", "apps/templates", "blue-green", "blue-green/templates", "guestbook", "helm-dependency",
				"helm-guestbook", "helm-guestbook/templates", "helm-hooks", "jsonnet-guestbook", "jsonnet-guestbook-tla",
				"ksonnet-guestbook", "ksonnet-guestbook/components", "ksonnet-guestbook/environments", "ksonnet-guestbook/environments/default",
				"ksonnet-guestbook/environments/dev", "ksonnet-guestbook/environments/prod", "kustomize-guestbook", "plugins", "plugins/kasane",
				"plugins/kustomized-helm", "plugins/kustomized-helm/overlays", "pre-post-sync", "sock-shop", "sock-shop/base", "sync-waves"},
		},
		{
			name:     "If GetRepository returns an error, it should pass back to caller",
			repoURL:  "https://github.com/argoproj/argocd-example-apps/",
			revision: exampleRepoRevision,
			repoRes: &v1alpha1.Repository{
				Repo: "https://github.com/argoproj/argocd-example-apps/",
			},
			repoErr:       errors.New("Simulated error from GetRepository"),
			expected:      nil,
			expectedError: errors.New("Error in GetRepository: Simulated error from GetRepository"),
		},
		{
			name: "Test against repository containing no directories",
			// Here I picked an arbitrary repository in argoproj-labs, with a commit containing no folders.
			repoURL:  "https://github.com/argoproj-labs/argo-workflows-operator/",
			revision: "5f50933a576833b73b7a172909d8545a108685f4",
			repoRes: &v1alpha1.Repository{
				Repo: "https://github.com/argoproj-labs/argo-workflows-operator/",
			},
			repoErr:  nil,
			expected: []string{},
		},
	} {
		cc := c
		t.Run(cc.name, func(t *testing.T) {
			argocdRepositoryMock := ArgocdRepositoryMock{mock: &mock.Mock{}}
			repoClientsetMock := repoClientsetMock{mock: &mock.Mock{}}

			argocdRepositoryMock.mock.On("GetRepository", mock.Anything, cc.repoURL).Return(cc.repoRes, cc.repoErr)

			argocd := argoCDService{
				repositoriesDB: argocdRepositoryMock,
				repoClientset:  repoClientsetMock,
			}

			got, err := argocd.GetDirectories(context.TODO(), cc.repoURL, cc.revision)

			if cc.expectedError != nil {
				assert.EqualError(t, err, cc.expectedError.Error())
			} else {
				sort.Strings(got)
				sort.Strings(cc.expected)

				assert.Equal(t, got, cc.expected)
				assert.NoError(t, err)
			}
		})
	}
}
