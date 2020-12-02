package generators

import (
	"context"
	"path"
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/services"
	log "github.com/sirupsen/logrus"
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

func (g *GitGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	return time.Duration(appSetGenerator.Git.RequeueAfterSeconds) * time.Second
}

func (g *GitGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {

	if appSetGenerator == nil {
		return nil, EmptyAppSetGeneratorError
	}

	if appSetGenerator.Git == nil {
		return nil, EmptyAppSetGeneratorError
	}

	allApps, err := g.repos.GetApps(context.TODO(), appSetGenerator.Git.RepoURL, appSetGenerator.Git.Revision)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"allAps":   allApps,
		"total":    len(allApps),
		"repoURL":  appSetGenerator.Git.RepoURL,
		"revision": appSetGenerator.Git.Revision,
	}).Info("applications result from the repo service")

	requestedApps := g.filter(appSetGenerator.Git.Directories, allApps)

	res := g.generateParams(requestedApps, appSetGenerator)

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

func (g *GitGenerator) generateParams(requestedApps []string, appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) []map[string]string {

	res := make([]map[string]string, len(requestedApps))
	for i, a := range requestedApps {

		params := make(map[string]string, 2)
		params["path"] = a
		params["path.basename"] = path.Base(a)

		res[i] = params
	}

	return res
}
