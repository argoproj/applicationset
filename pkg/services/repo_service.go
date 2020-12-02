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

// RepositoryDB Is a lean facade for ArgoDB,
// Using a lean interface makes it more easy to test the functionality the git generator uses
type RepositoryDB interface {
	GetRepository(ctx context.Context, url string) (*v1alpha1.Repository, error)
}

type argoCDService struct {
	repositoriesDB RepositoryDB
	repoClientset  apiclient.Clientset
}

type Apps interface {
	GetApps(ctx context.Context, repoURL string, revision string) ([]string, error)
}

func NewArgoCDService(ctx context.Context, clientset kubernetes.Interface, namespace string, repoServerAddress string) Apps {
	settingsMgr := settings.NewSettingsManager(ctx, clientset, namespace)

	return &argoCDService{
		repositoriesDB: db.NewDB(namespace, settingsMgr, clientset).(RepositoryDB),
		repoClientset:  apiclient.NewRepoServerClientset(repoServerAddress, 5),
	}
}

func (a *argoCDService) GetApps(ctx context.Context, repoURL string, revision string) ([]string, error) {
	repo, err := a.repositoriesDB.GetRepository(ctx, repoURL)
	if err != nil {

		return nil, errors.Wrap(err, "Error in GetRepository")
	}

	conn, repoClient, err := a.repoClientset.NewRepoServerClient()
	defer io.Close(conn)
	if err != nil {
		return nil, errors.Wrap(err, "Error in creating repo service client")
	}

	apps, err := repoClient.ListApps(ctx, &apiclient.ListAppsRequest{
		Repo:     repo,
		Revision: revision,
	})
	log.Debugf("apps - %#v", apps)
	if err != nil {
		return nil, errors.Wrap(err, "Error in ListApps")
	}

	res := []string{}

	for name, _ := range apps.Apps {
		res = append(res, name)
	}

	return res, nil
}
