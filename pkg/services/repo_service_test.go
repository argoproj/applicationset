package services

import (
	"context"
	"errors"
	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/reposerver/apiclient"
	"github.com/argoproj/gitops-engine/pkg/utils/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"testing"
)

type ArgocdRepositoryMock struct {
	mock.Mock
}

func (a ArgocdRepositoryMock) GetRepository(ctx context.Context, url string) (*v1alpha1.Repository, error) {
	args := a.Called(ctx, url)

	return args.Get(0).(*v1alpha1.Repository), args.Error(1)

}

type repoServerClientMock struct {
	mock.Mock
}

func (r repoServerClientMock) GenerateManifest(ctx context.Context, in *apiclient.ManifestRequest, opts ...grpc.CallOption) (*apiclient.ManifestResponse, error) {
	return nil, nil
}
func (r repoServerClientMock) ListApps(ctx context.Context, in *apiclient.ListAppsRequest, opts ...grpc.CallOption) (*apiclient.AppList, error) {
	args := r.Called(ctx, in)

	return args.Get(0).(*apiclient.AppList), args.Error(1)
}
func (r repoServerClientMock) GetAppDetails(ctx context.Context, in *apiclient.RepoServerAppDetailsQuery, opts ...grpc.CallOption) (*apiclient.RepoAppDetailsResponse, error) {
	return nil, nil
}
func (r repoServerClientMock) GetRevisionMetadata(ctx context.Context, in *apiclient.RepoServerRevisionMetadataRequest, opts ...grpc.CallOption) (*v1alpha1.RevisionMetadata, error) {
	return nil, nil
}
func (r repoServerClientMock) GetHelmCharts(ctx context.Context, in *apiclient.HelmChartsRequest, opts ...grpc.CallOption) (*apiclient.HelmChartsResponse, error) {
	return nil, nil
}

type closer struct {
	mock.Mock
}

func (c closer) Close() error {
	return nil
}

type repoClientsetMock struct {
	mock.Mock
}

func (r repoClientsetMock) NewRepoServerClient() (io.Closer, apiclient.RepoServerServiceClient, error) {
	args := r.Called()

	return closer{}, args.Get(0).(apiclient.RepoServerServiceClient), args.Error(1)
}

func TestGetApps(t *testing.T) {

	for _, c := range []struct {
		name          string
		repoURL       string
		revision      string
		repoRes       *v1alpha1.Repository
		repoErr       error
		appRes        *apiclient.AppList
		appError      error
		expected      []string
		expectedError error
	}{
		{
			"Happy Flow",
			"repoURL",
			"revision",
			&v1alpha1.Repository{},
			nil,
			&apiclient.AppList{
				Apps: map[string]string{
					"app1": "",
					"app2": "",
				},
			},
			nil,
			[]string{"app1", "app2"},
			nil,
		},
		{
			"handles GetRepository error",
			"repoURL",
			"revision",
			&v1alpha1.Repository{},
			errors.New("error"),
			&apiclient.AppList{
				Apps: map[string]string{
					"app1": "",
					"app2": "",
				},
			},
			nil,
			[]string{},
			errors.New("Error in GetRepository: error"),
		},
		{
			"handles ListApps error",
			"repoURL",
			"revision",
			&v1alpha1.Repository{},
			nil,
			&apiclient.AppList{
				Apps: map[string]string{
					"app1": "",
					"app2": "",
				},
			},
			errors.New("error"),
			[]string{},
			errors.New("Error in ListApps: error"),
		},
	} {
		cc := c
		t.Run(cc.name, func(t *testing.T) {
			argocdRepositoryMock := ArgocdRepositoryMock{}
			repoServerClientMock := repoServerClientMock{}
			repoClientsetMock := repoClientsetMock{}

			argocdRepositoryMock.On("GetRepository", mock.Anything, cc.repoURL).Return(cc.repoRes, cc.repoErr)

			repoServerClientMock.On("ListApps", mock.Anything, &apiclient.ListAppsRequest{
				Repo:     cc.repoRes,
				Revision: cc.revision,
			}).Return(cc.appRes, cc.appError)

			repoClientsetMock.On("NewRepoServerClient").Return(repoServerClientMock, nil)

			argocd := argoCDService{
				repositoriesDB: argocdRepositoryMock,
				repoClientset:  repoClientsetMock,
			}
			got, err := argocd.GetApps(context.TODO(), cc.repoURL, cc.revision)

			if cc.expectedError != nil {
				assert.EqualError(t, err, cc.expectedError.Error())
			} else {
				assert.Equal(t, got, cc.expected)
				assert.NoError(t, err)
			}
		})
	}
}
