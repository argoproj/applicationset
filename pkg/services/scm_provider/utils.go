package scm_provider

import (
	"context"
	"fmt"
	"regexp"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
)

func compileFilters(filters []argoprojiov1alpha1.SCMProviderGeneratorFilter) ([]*Filter, error) {
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
		if filter.PathsExist != nil {
			outFilter.PathsExist = filter.PathsExist
		}
		if filter.BranchMatch != nil {
			outFilter.BranchMatch, err = regexp.Compile(*filter.BranchMatch)
			if err != nil {
				return nil, fmt.Errorf("error compiling BranchMatch regexp %q: %v", *filter.LabelMatch, err)
			}
		}
		outFilters = append(outFilters, outFilter)
	}
	return outFilters, nil
}

func matchFilter(ctx context.Context, provider SCMProviderService, repo *Repository, filter *Filter) (bool, error) {
	if filter.RepositoryMatch != nil && !filter.RepositoryMatch.MatchString(repo.Repository) {
		return false, nil
	}

	if filter.BranchMatch != nil && !filter.BranchMatch.MatchString(repo.Branch) {
		return false, nil
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
			return false, nil
		}
	}

	if len(filter.PathsExist) != 0 {
		for _, path := range filter.PathsExist {
			hasPath, err := provider.RepoHasPath(ctx, repo, path)
			if err != nil {
				return false, err
			}
			if !hasPath {
				return false, nil
			}
		}
	}

	return true, nil
}

func ListRepos(ctx context.Context, provider SCMProviderService, filters []argoprojiov1alpha1.SCMProviderGeneratorFilter, cloneProtocol string) ([]*Repository, error) {
	compiledFilters, err := compileFilters(filters)
	if err != nil {
		return nil, err
	}

	repos, err := provider.ListRepos(ctx, cloneProtocol)
	if err != nil {
		return nil, err
	}

	// Special case, if we have no filters, allow everything.
	if len(compiledFilters) == 0 {
		return repos, nil
	}

	filteredRepos := make([]*Repository, 0, len(repos))
	for _, repo := range repos {
		for _, filter := range compiledFilters {
			matches, err := matchFilter(ctx, provider, repo, filter)
			if err != nil {
				return nil, err
			}
			if matches {
				filteredRepos = append(filteredRepos, repo)
				break
			}
		}
	}
	return filteredRepos, nil
}
