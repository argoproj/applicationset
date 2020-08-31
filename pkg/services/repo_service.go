package services

import (
	"context"
	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/reposerver/apiclient"
	"github.com/argoproj/argo-cd/util/db"
	"github.com/argoproj/argo-cd/util/settings"
	"github.com/argoproj/gitops-engine/pkg/utils/io"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

type ArgocdRepository interface {
	GetRepository(ctx context.Context, url string) (*v1alpha1.Repository, error)
}

type ArgoCDService struct {
	ArgocdRepository 	ArgocdRepository
	repoClientset 		apiclient.Clientset
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

	argocdDB := db.NewDB(namespace, settingsMgr, clientset)

	return &ArgoCDService{ArgocdRepository: argocdDB.(ArgocdRepository), repoClientset: repoClientset}, nil
}

func (a *ArgoCDService) GetApps(ctx context.Context, repoURL string, revision string, path string) ([]string, error) {
	repo, err := a.ArgocdRepository.GetRepository(ctx, repoURL)
	if err != nil {

		return nil, errors.Wrap(err, "Error in GetRepository")
	}
	log.Infof("repo - %#v", repo)

	conn, repoClient, err := a.repoClientset.NewRepoServerClient()
	defer io.Close(conn)
	if err != nil {
		return nil, err
	}

	apps, err := repoClient.ListApps(ctx, &apiclient.ListAppsRequest{
		Repo: repo,
		Revision: revision,
		//Path: GitGenerator.Directories,
	})
	log.Infof("apps - %#v", apps)
	if err != nil {
		return nil, errors.Wrap(err, "Error in ListApps")
	}

	res := []string{}

	for name, _ := range apps.Apps {
		res = append(res, name)
	}

	return res, nil
}
