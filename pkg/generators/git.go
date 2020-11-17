package generators

import (
	"context"
	"encoding/json"
	"path"
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/services"
	"github.com/jeremywohl/flatten"
	log "github.com/sirupsen/logrus"
)

var _ Generator = (*GitGenerator)(nil)

type GitGenerator struct {
	repos services.Repos
}

func NewGitGenerator(repos services.Repos) Generator {
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

	var err error
	var res []map[string]string
	if appSetGenerator.Git.Directories != nil {
		res, err = g.generateParamsForGitDirectories(appSetGenerator)
	} else if appSetGenerator.Git.Files != nil {
		res, err = g.generateParamsForGitFiles(appSetGenerator)
	} else {
		return nil, EmptyAppSetGeneratorError
	}
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (g *GitGenerator) generateParamsForGitDirectories(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {
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

	requestedApps := g.filterApps(appSetGenerator.Git.Directories, allApps)

	res := g.generateParamsFromApps(requestedApps, appSetGenerator)

	return res, nil
}

func (g *GitGenerator) generateParamsForGitFiles(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {
	allPaths := []string{}
	for _, requestedPath := range appSetGenerator.Git.Files {
		paths, err := g.repos.GetPaths(context.TODO(), appSetGenerator.Git.RepoURL, appSetGenerator.Git.Revision, requestedPath.Path)
		if err != nil {
			return nil, err
		}
		allPaths = append(allPaths, paths...)
	}

	res := []map[string]string{}

	for _, path := range allPaths {
		params, err := g.generateParamsFromGitFile(appSetGenerator, path)
		if err != nil {
			return nil, err
		}

		res = append(res, params)
	}
	return res, nil
}

func (g *GitGenerator) generateParamsFromGitFile(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator, path string) (map[string]string, error) {
	content, err := g.repos.GetFileContent(context.TODO(), appSetGenerator.Git.RepoURL, appSetGenerator.Git.Revision, path)
	if err != nil {
		return nil, err
	}

	config := make(map[string]interface{})
	err = json.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	flat, err := flatten.Flatten(config, "", flatten.DotStyle)
	if err != nil {
		return nil, err
	}
	params := make(map[string]string)
	for k, v := range flat {
		params[k] = v.(string)
	}

	return params, nil
}

func (g *GitGenerator) filterApps(Directories []argoprojiov1alpha1.GitDirectoryGeneratorItem, allApps []string) []string {
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

func (g *GitGenerator) generateParamsFromApps(requestedApps []string, _ *argoprojiov1alpha1.ApplicationSetGenerator) []map[string]string {
	// TODO: At some point, the appicationSetGenerator param should be used

	res := make([]map[string]string, len(requestedApps))
	for i, a := range requestedApps {

		params := make(map[string]string, 2)
		params["path"] = a
		params["path.basename"] = path.Base(a)

		res[i] = params
	}

	return res
}
