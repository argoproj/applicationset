package services

import (
	"context"
	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/reposerver/apiclient"
	"github.com/argoproj/argo-cd/util"
	"github.com/argoproj/argo-cd/util/db"
	"github.com/argoproj/argo-cd/util/settings"
	"k8s.io/client-go/kubernetes"
)

type ArgocdRepository interface {
	GetRepository(ctx context.Context, url string) (*v1alpha1.Repository, error)
}

type ArgoCDService struct {
	ArgocdRepository ArgocdRepository
	repoServerClient apiclient.RepoServerServiceClient
	closer          util.Closer
}

type Apps struct {
	name	string
}

type Repos interface {
	GetApps(ctx context.Context, repoURL string, revision string, path string) ([]string, error)
}

func NewArgoCDService(ctx context.Context, clientset kubernetes.Interface, namespace string, repoServerAddress string) (*ArgoCDService, error) {
	settingsMgr := settings.NewSettingsManager(ctx, clientset, namespace)
	repoClientset := apiclient.NewRepoServerClientset(repoServerAddress, 5)
	closer, repoClient, err := repoClientset.NewRepoServerClient()
	if err != nil {
		return nil, err
	}

	argocdDB := db.NewDB(namespace, settingsMgr, clientset)

	return &ArgoCDService{ArgocdRepository: argocdDB.(ArgocdRepository), repoServerClient: repoClient, closer: closer}, nil
}

func (a *ArgoCDService) GetApps(ctx context.Context, repoURL string, revision string, path string) ([]string, error) {
	defer a.closer.Close()
	repo, err := a.ArgocdRepository.GetRepository(ctx, repoURL)
	if err != nil {
		return nil, err
	}

	apps, err := a.repoServerClient.ListApps(ctx, &apiclient.ListAppsRequest{
		Repo: repo,
		Revision: revision,
		//Path: GitGenerator.Directories,
	})
	if err != nil {
		return nil, err
	}

	res := []string{}

	for name, _ := range apps.Apps {
		res = append(res, name)
	}

	return res, nil
}
