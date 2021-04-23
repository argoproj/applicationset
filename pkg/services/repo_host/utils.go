package repo_host

import (
	"context"
	"fmt"
	"regexp"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
)

func compileFilters(filters []argoprojiov1alpha1.RepoHostGeneratorFilter) ([]*Filter, error) {
	outFilters := make([]*Filter, 0, len(filters))
	for _, filter := range filters {
		outFilter := &Filter{}
		var err error
		if filter.RepositoryMatch != nil {
			outFilter.RepositoryMatch, err = regexp.Compile(*filter.RepositoryMatch)
			if err != nil {
				return nil, fmt.Errorf("error compiling RepositoryMatch regexp %q: %v", *filter.RepositoryMatch, err)
			}
		}
		if filter.LabelMatch != nil {
			outFilter.LabelMatch, err = regexp.Compile(*filter.LabelMatch)
			if err != nil {
				return nil, fmt.Errorf("error compiling LabelMatch regexp %q: %v", *filter.LabelMatch, err)
			}
		}
		if filter.PathExists != nil {
			outFilter.PathExists = filter.PathExists
		}
		outFilters = append(outFilters, outFilter)
	}
	return outFilters, nil
}

func ListRepos(ctx context.Context, host RepoHostService, filters []argoprojiov1alpha1.RepoHostGeneratorFilter) ([]*HostedRepo, error) {
	compiledFilters, err := compileFilters(filters)
	if err != nil {
		return nil, err
	}

	repos, err := host.ListRepos(ctx)
	if err != nil {
		return nil, err
	}
	filteredRepos := make([]*HostedRepo, 0, len(repos))
	for _, repo := range repos {
		matches := true
		for _, filter := range compiledFilters {
			if filter.RepositoryMatch != nil {
				if !filter.RepositoryMatch.MatchString(repo.Repository) {
					matches = false
					break
				}
			}

			if filter.LabelMatch != nil {
				found := false
				for _, label := range repo.Labels {
					if filter.LabelMatch.MatchString(label) {
						found = true
						break
					}
				}
				if !found {
					matches = false
					break
				}
			}

			if filter.PathExists != nil {
				hasPath, err := host.RepoHasPath(ctx, repo, *filter.PathExists)
				if err != nil {
					return nil, err
				}
				if !hasPath {
					matches = false
					break
				}
			}
		}
		if matches {
			filteredRepos = append(filteredRepos, repo)
		}
	}
	return filteredRepos, nil
}
