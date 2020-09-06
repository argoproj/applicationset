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

	allApps, err := g.repos.GetApps(context.TODO(), appSetGenerator.Git.RepoURL, appSetGenerator.Git.Revision)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"allAps": allApps,
		"total": len(allApps),
		"repoURL": appSetGenerator.Git.RepoURL,
		"revision": appSetGenerator.Git.Revision,
	}).Info("applications result from the repo service")

	requestedApps := g.filter(appSetGenerator.Git.Directories, allApps)

	res := g.generateApplications(requestedApps, appSet)

	return res, nil
}

func (g *GitGenerator) filter(Directories []argoprojiov1alpha1.GitDirectoryGeneratorItem, allApps []string) []string {
	res := []string{}
	for _, requestedPath := range Directories {
		for _, appPath := range allApps {
			match, err := path.Match(requestedPath.Path, appPath)
			if err != nil {
				log.WithError(err).WithField("requestedPath", requestedPath).
					WithField("appPath", appPath).Error("error while matching appPath to requestedPath")
				continue
			}
			if match {
				res = append(res, appPath)
			}
		}
	}
	return res
}

func (g *GitGenerator) generateApplications(requestedApps []string, appSet *argoprojiov1alpha1.ApplicationSet) ([]argov1alpha1.Application) {

	res := make([]argov1alpha1.Application, len(requestedApps))
	for i, a := range requestedApps {
		app, err := g.generateApplication(appSet, a)
		if err != nil {
			log.WithError(err).WithField("app", a).Error("error while generating app")
			continue
		}
		res[i] = *app
	}

	return res
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
