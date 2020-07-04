package generators

import (
	"context"
	"fmt"
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/utils"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/reposerver/apiclient"
	log "github.com/sirupsen/logrus"
	"path"
)

var _ Generator = (*GitGenerator)(nil)

type GitGenerator struct {
	repoClientset apiclient.Clientset
}

func NewGitGenerator(repoClientset apiclient.Clientset) Generator {
	//repoClientset := apiclient.NewRepoServerClientset(repoServerAddress, 5)
	g := &GitGenerator{
		repoClientset: repoClientset,
	}
	return g
}

func (g *GitGenerator) GenerateApplications(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator, appSet *argoprojiov1alpha1.ApplicationSet) ([]argov1alpha1.Application, error) {
	if appSetGenerator == nil || appSet == nil {
		return nil, fmt.Errorf("ApplicationSet is empty")
	}

	if appSetGenerator.Git == nil {
		return nil, fmt.Errorf("git variable empty")
	}

	if len(appSetGenerator.Git.Directories) > 0 {
		paths, err := g.GetAllPaths(appSetGenerator.Git)
		if err != nil {
			return nil, err
		}

		res := make([]argov1alpha1.Application, len(paths))
		for i, p := range paths {
			app, err := g.generateApplicationFromPath(appSet, p)
			if err != nil {
				log.WithError(err).WithField("path", p).Error("error while generating app from path")
				continue
			}
			res[i] = *app
		}

		return res, nil
	}

	return nil, nil
}

func (g *GitGenerator) generateApplicationFromPath(appSet *argoprojiov1alpha1.ApplicationSet, appPath string) (*argov1alpha1.Application, error) {
	var tmplApplication argov1alpha1.Application
	tmplApplication.Namespace = appSet.Spec.Template.Namespace
	tmplApplication.Name = appSet.Spec.Template.Name
	tmplApplication.Spec = appSet.Spec.Template.Spec

	params := make(map[string]string, 2)
	params["path"] = appPath
	params["path.basename"] = path.Base(appPath)

	tmpApplication, err := utils.RenderTemplateParams(&tmplApplication, params)

	return tmpApplication, err
}

func (g *GitGenerator) GetAllPaths(GitGenerator *argoprojiov1alpha1.GitGenerator) ([]string, error) {
	closer, repoClient, err := g.repoClientset.NewRepoServerClient()
	defer closer.Close()
	if err != nil {
		return nil, err
	}

	listAppsRequest := &apiclient.ListAppsRequest{
		Repo: &argov1alpha1.Repository{
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
