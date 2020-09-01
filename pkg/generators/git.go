package generators

import (
	"context"
	"fmt"
	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/services"
	"github.com/argoproj-labs/applicationset/pkg/utils"
	argov1alpha1 "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	log "github.com/sirupsen/logrus"
	"path"
)

var _ Generator = (*GitGenerator)(nil)

type GitGenerator struct {
	repos services.Apps
}

func NewGitGenerator(repos services.Apps) Generator {
	g := &GitGenerator{
		repos: repos,
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

	res := []argov1alpha1.Application{}

	for _, path := range appSetGenerator.Git.Directories {
		apps, err := g.generateApplications(appSetGenerator.Git.RepoURL, appSetGenerator.Git.Revision, path.Path, appSet)
		if err != nil {
			log.WithError(err).WithField("path", path).Error("error while generating app from path")
			continue
		}

		res = append(res, apps...)
	}

	return res, nil
}

func (g *GitGenerator) generateApplications(repoURL, revision, path string, appSet *argoprojiov1alpha1.ApplicationSet) ([]argov1alpha1.Application, error) {
	appsPath, err := g.repos.GetApps(context.TODO(), repoURL, revision, path)
	if err != nil {
		return nil, err
	}

	res := make([]argov1alpha1.Application, len(appsPath))
	for i, a := range appsPath {
		app, err := g.generateApplication(appSet, a)
		if err != nil {
			log.WithError(err).WithField("path", path).Error("error while generating app from path")
			continue
		}
		res[i] = *app
	}

	return res, nil
}


func (g *GitGenerator) generateApplication(appSet *argoprojiov1alpha1.ApplicationSet, appPath string) (*argov1alpha1.Application, error) {
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
