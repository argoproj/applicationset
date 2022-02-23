package scm_provider

import "context"

type MockProvider struct {
	Repos           []*Repository
	allPullRequests bool
}

var _ SCMProviderService = &MockProvider{}

func (m *MockProvider) ListRepos(_ context.Context, _ string) ([]*Repository, error) {
	repos := []*Repository{}
	for _, candidateRepo := range m.Repos {
		found := false
		for _, alreadySetRepo := range repos {
			if alreadySetRepo.Repository == candidateRepo.Repository {
				found = true
				break
			}
		}
		if !found {
			repos = append(repos, candidateRepo)
		}
	}
	return repos, nil
}

func (*MockProvider) RepoHasPath(_ context.Context, repo *Repository, path string) (bool, error) {
	return path == repo.Repository, nil
}

func (m *MockProvider) GetBranches(_ context.Context, repo *Repository) ([]*Repository, error) {
	branchRepos := []*Repository{}
	for _, candidateRepo := range m.Repos {
		if candidateRepo.Repository == repo.Repository {
			found := false
			for _, alreadySetRepo := range branchRepos {
				if alreadySetRepo.Branch == candidateRepo.Branch {
					found = true
					break
				}
			}
			if !found {
				branchRepos = append(branchRepos, candidateRepo)
			}
		}
	}
	return branchRepos, nil
}

func (m *MockProvider) GetPullRequests(_ context.Context, repo *Repository) ([]*Repository, error) {
	pullRequestRepos := []*Repository{}
	if !m.allPullRequests {
		return pullRequestRepos, nil
	}
	for _, candidateRepo := range m.Repos {
		if candidateRepo.Repository == repo.Repository {
			found := false
			for _, alreadySetRepo := range pullRequestRepos {
				if alreadySetRepo.Branch == candidateRepo.Branch {
					found = true
					break
				}
			}
			if !found {
				pullRequestRepos = append(pullRequestRepos, candidateRepo)
			}
		}
	}
	return pullRequestRepos, nil
}
