package scm_provider

import "context"

type MockProvider struct {
	Repos []*Repository
}

var _ SCMProviderService = &MockProvider{}

func (m *MockProvider) ListRepos(_ context.Context, _ string) ([]*Repository, error) {
	return m.Repos, nil
}

func (*MockProvider) RepoHasPath(_ context.Context, repo *Repository, path string) (bool, error) {
	return path == repo.Repository, nil
}
