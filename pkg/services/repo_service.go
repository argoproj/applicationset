package services

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/reposerver/apiclient"
	"github.com/argoproj/argo-cd/util/db"
	"github.com/argoproj/argo-cd/util/git"
	"github.com/argoproj/argo-cd/util/io"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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

type Repos interface {
	// GetApps return a list of valid Argo CD Application sources within the repo, as per the rules described here: https://argoproj.github.io/argo-cd/user-guide/tool_detection/
	GetApps(ctx context.Context, repoURL string, revision string) ([]string, error)

	// GetPaths returns a list of files (not directories) within the target repo
	GetPaths(ctx context.Context, repoURL string, revision string, pattern string) ([]string, error)

	// GetDirectories returns a list of directories (not files) within the target repo
	GetDirectories(ctx context.Context, repoURL string, revision string) ([]string, error)

	// GetFileContent returns the contents of a particular repository file
	GetFileContent(ctx context.Context, repoURL string, revision string, path string) ([]byte, error)
}

func NewArgoCDService(db db.ArgoDB, repoServerAddress string) Repos {

	return &argoCDService{
		repositoriesDB: db.(RepositoryDB),
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

	for name := range apps.Apps {
		res = append(res, name)
	}

	return res, nil
}

func (a *argoCDService) GetPaths(ctx context.Context, repoURL string, revision string, pattern string) ([]string, error) {
	repo, err := a.repositoriesDB.GetRepository(ctx, repoURL)
	if err != nil {
		return nil, errors.Wrap(err, "Error in GetRepository")
	}

	gitRepoClient, err := git.NewClient(repo.Repo, repo.GetGitCreds(), repo.IsInsecure(), repo.IsLFSEnabled())

	if err != nil {
		return nil, err
	}

	err = checkoutRepo(gitRepoClient, revision)
	if err != nil {
		return nil, err
	}

	paths, err := gitRepoClient.LsFiles(pattern)
	if err != nil {
		return nil, errors.Wrap(err, "Error during listing files of local repo")
	}

	return paths, nil
}

func (a *argoCDService) GetDirectories(ctx context.Context, repoURL string, revision string) ([]string, error) {

	repo, err := a.repositoriesDB.GetRepository(ctx, repoURL)
	if err != nil {
		return nil, errors.Wrap(err, "Error in GetRepository")
	}

	gitRepoClient, err := git.NewClient(repo.Repo, repo.GetGitCreds(), repo.IsInsecure(), repo.IsLFSEnabled())
	if err != nil {
		return nil, err
	}

	err = checkoutRepo(gitRepoClient, revision)
	if err != nil {
		return nil, err
	}

	filteredPaths := []string{}

	repoRoot := gitRepoClient.Root()

	if err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, fnErr error) error {
		if fnErr != nil {
			return fnErr
		}
		if !info.IsDir() { // Skip files: directories only
			return nil
		}

		fname := info.Name()
		if fname == ".git" { // Skip repository metadata
			return filepath.SkipDir
		}

		relativePath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}

		if relativePath == "." { // Exclude '.' from results
			return nil
		}

		filteredPaths = append(filteredPaths, relativePath)

		return nil
	}); err != nil {
		return nil, err
	}

	return filteredPaths, nil

}

func (a *argoCDService) GetFileContent(ctx context.Context, repoURL string, revision string, path string) ([]byte, error) {
	repo, err := a.repositoriesDB.GetRepository(ctx, repoURL)
	if err != nil {
		return nil, errors.Wrap(err, "Error in GetRepository")
	}

	gitRepoClient, err := git.NewClient(repo.Repo, repo.GetGitCreds(), repo.IsInsecure(), repo.IsLFSEnabled())

	if err != nil {
		return nil, err
	}

	err = checkoutRepo(gitRepoClient, revision)
	if err != nil {
		return nil, err
	}

	bytes, err := ioutil.ReadFile(filepath.Join(gitRepoClient.Root(), path))
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func checkoutRepo(gitRepoClient git.Client, revision string) error {
	err := gitRepoClient.Init()
	if err != nil {
		return errors.Wrap(err, "Error during initializing repo")
	}

	err = gitRepoClient.Fetch()
	if err != nil {
		return errors.Wrap(err, "Error during fetching repo")
	}

	commitSHA, err := gitRepoClient.LsRemote(revision)
	if err != nil {
		return errors.Wrap(err, "Error during fetching commitSHA")
	}
	err = gitRepoClient.Checkout(commitSHA)
	if err != nil {
		return errors.Wrap(err, "Error during repo checkout")
	}
	return nil
}
