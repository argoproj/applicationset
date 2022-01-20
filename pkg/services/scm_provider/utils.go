package scm_provider

import (
	"context"
	"fmt"
	"regexp"

	argoprojiov1alpha1 "github.com/argoproj/applicationset/api/v1alpha1"
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

	repoFilters := getApplicableFilters(compiledFilters, false)
	if len(repoFilters) == 0 {
		repos, err := getBranches(ctx, provider, repos, compiledFilters)
		if err != nil {
			return nil, err
		}
		return repos, nil
	}

	filteredRepos := make([]*Repository, 0, len(repos))
	for _, repo := range repos {
		for _, filter := range repoFilters {
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

	repos, err = getBranches(ctx, provider, filteredRepos, compiledFilters)
	if err != nil {
		return nil, err
	}
	return repos, nil
}

func getBranches(ctx context.Context, provider SCMProviderService, repos []*Repository, compiledFilters []*Filter) ([]*Repository, error) {
	reposWithBranches := []*Repository{}
	for _, repo := range repos {
		reposFilled, err := provider.GetBranches(ctx, repo)
		if err != nil {
			return nil, err
		}
		reposWithBranches = append(reposWithBranches, reposFilled...)
	}
	branchFilters := getApplicableFilters(compiledFilters, true)
	if len(branchFilters) == 0 {
		return reposWithBranches, nil
	}
	filteredRepos := make([]*Repository, 0, len(reposWithBranches))
	for _, repo := range reposWithBranches {
		for _, filter := range branchFilters {
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

// getApplicableFilters returns filters which either have a branch filter or do not have branch filters, depending on the hasBranchFilter argument.
func getApplicableFilters(filters []*Filter, branchFilter bool) []*Filter {
	applicableFilters := []*Filter{}
	for _, filter := range filters {
		if branchFilter && (filter.BranchMatch != nil || filter.PathsExist != nil) {
			applicableFilters = append(applicableFilters, filter)
		} else if !branchFilter && filter.BranchMatch == nil && filter.PathsExist == nil {
			applicableFilters = append(applicableFilters, filter)
		}
	}
	return applicableFilters
}
