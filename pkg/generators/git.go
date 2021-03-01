package generators

import (
	"context"
	"encoding/json"
	"path"
	"sort"
	"time"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/services"
	"github.com/jeremywohl/flatten"
	log "github.com/sirupsen/logrus"
)

var _ Generator = (*GitGenerator)(nil)

const (
	DefaultRequeueAfterSeconds = 3 * time.Minute
)

type GitGenerator struct {
	repos services.Repos
}

func NewGitGenerator(repos services.Repos) Generator {
	g := &GitGenerator{
		repos: repos,
	}
	return g
}

func (g *GitGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return &appSetGenerator.Git.Template
}

func (g *GitGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {

	// Return a requeue default of 3 minutes, if no default is specified.

	if appSetGenerator.Git.RequeueAfterSeconds != nil {
		return time.Duration(*appSetGenerator.Git.RequeueAfterSeconds) * time.Second
	}

	return DefaultRequeueAfterSeconds
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

	// Directories, not files
	allPaths, err := g.repos.GetDirectories(context.TODO(), appSetGenerator.Git.RepoURL, appSetGenerator.Git.Revision)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"allPaths": allPaths,
		"total":    len(allPaths),
		"repoURL":  appSetGenerator.Git.RepoURL,
		"revision": appSetGenerator.Git.Revision,
	}).Info("applications result from the repo service")

	requestedApps := g.filterApps(appSetGenerator.Git.Directories, allPaths)

	res := g.generateParamsFromApps(requestedApps, appSetGenerator)

	return res, nil
}

func (g *GitGenerator) generateParamsForGitFiles(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) ([]map[string]string, error) {

	// Get all paths that match the requested path string, removing duplicates
	allPathsMap := make(map[string]bool)
	for _, requestedPath := range appSetGenerator.Git.Files {
		paths, err := g.repos.GetFilePaths(context.TODO(), appSetGenerator.Git.RepoURL, appSetGenerator.Git.Revision, requestedPath.Path)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			allPathsMap[path] = true
		}
	}

	// Extract the unduplicated map into a list, and sort by path to ensure a deterministic
	// processing order in the subsequent step
	allPaths := []string{}
	for path := range allPathsMap {
		allPaths = append(allPaths, path)
	}
	sort.Strings(allPaths)

	// Generate params from each path, and return
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
