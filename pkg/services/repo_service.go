package services

import (
	"context"
	"github.com/argoproj/argo-cd/reposerver/apiclient"
	"github.com/argoproj/argo-cd/util/db"
	"github.com/argoproj/argo-cd/util/settings"
	"k8s.io/client-go/kubernetes"
	log "github.com/sirupsen/logrus"
)

type argoCDService struct {
	clientset        kubernetes.Interface
	namespace        string
	settingsMgr      *settings.SettingsManager
	repoServerClient apiclient.RepoServerServiceClient
	dispose          func()
}

type Apps struct {
	name	string
}

type Repos interface {
	GetApps(ctx context.Context, repoURL string, revision string, path string) ([]string, error)
}

func NewArgoCDService(clientset kubernetes.Interface, namespace string, repoServerAddress string) (*argoCDService, error) {
	ctx, cancel := context.WithCancel(context.Background())
	settingsMgr := settings.NewSettingsManager(ctx, clientset, namespace)
	repoClientset := apiclient.NewRepoServerClientset(repoServerAddress, 5)
	closer, repoClient, err := repoClientset.NewRepoServerClient()
	if err != nil {
		cancel()
		return nil, err
	}

	dispose := func() {
		cancel()
		if err := closer.Close(); err != nil {
			log.Warnf("Failed to close repo server connection: %v", err)
		}
	}
	return &argoCDService{settingsMgr: settingsMgr, namespace: namespace, repoServerClient: repoClient, dispose: dispose}, nil
}

func (a *argoCDService) GetApps(ctx context.Context, repoURL string, revision string, path string) ([]string, error) {
	argocdDB := db.NewDB(a.namespace, a.settingsMgr, a.clientset)
	repo, err := argocdDB.GetRepository(ctx, repoURL)
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
