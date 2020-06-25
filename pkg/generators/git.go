package generators

import (
	"context"
	"fmt"
	"path"

	"github.com/argoproj-labs/applicationset/pkg/utils"
	argocdv1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/reposerver/apiclient"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	log "github.com/sirupsen/logrus"
)

var _ Generator = (*GitGenerator)(nil)

type GitGenerator struct {
	repoClientset apiclient.Clientset
}

func NewGitGenerator(repoServerAddress string) Generator {
	repoClientset := apiclient.NewRepoServerClientset(repoServerAddress, 5)
	g := &GitGenerator{
		repoClientset: repoClientset,
	}
	return g
}

func (g *GitGenerator) GenerateApplications(appSet *argoprojiov1alpha1.ApplicationSet) ([]argov1alpha1.Application, error) {
	if appSet == nil {
		return nil, fmt.Errorf("ApplicationSet is empty")
	}

	var GitGenerator *argoprojiov1alpha1.GitGenerator

	for _, tmpGenerator := range appSet.Spec.Generators {
		if tmpGenerator.Git != nil {
			GitGenerator = tmpGenerator.Git
			break
		}
	}

	if GitGenerator == nil {
		return nil, fmt.Errorf("There isn't git generator ")
	}

	if len(GitGenerator.Directories) > 0 {
		paths, err := g.GetAllPaths(GitGenerator)
		if err != nil {
			return nil, err
		}

		for _, p := range paths {

			var tmplApplication argov1alpha1.Application
			tmplApplication.Namespace = appSet.Spec.Template.Namespace
			tmplApplication.Name = appSet.Spec.Template.Name
			tmplApplication.Spec = appSet.Spec.Template.Spec

			params := make(map[string]string, 2)
			params["path"] = p
			params["path.basename"] = path.Base(p)

			tmpApplication, err := utils.RenderTemplateParams(&tmplApplication, params)
			log.Infof("tmpApplication %++v", tmpApplication)
			log.Infof("error %v", err)

		}

	}

	return nil, nil
}

func (g *GitGenerator) GetAllPaths(GitGenerator *argoprojiov1alpha1.GitGenerator) ([]string, error) {
	closer, repoClient, err := g.repoClientset.NewRepoServerClient()
	defer closer.Close()
	if err != nil {
		return nil, err
	}

	listAppsRequest := &apiclient.ListAppsRequest{
		Repo: &argocdv1alpha1.Repository{
			Repo: GitGenerator.RepoURL,
		},
		Revision: GitGenerator.Revision,
		//Path: GitGenerator.Directories,
	}

	appList, err := repoClient.ListApps(context.TODO(), listAppsRequest)
	if err != nil {
		return nil, err
	}

	var res []string

	for name, _ := range appList.Apps {
		res = append(res, name)
	}

	return res, nil
}
