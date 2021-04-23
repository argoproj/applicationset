package repo_host

import "context"

type MockRepoHost struct {
	Repos []*HostedRepo
}

var _ RepoHostService = &MockRepoHost{}

func (m *MockRepoHost) ListRepos(_ context.Context) ([]*HostedRepo, error) {
	return m.Repos, nil
}

func (*MockRepoHost) RepoHasPath(_ context.Context, repo *HostedRepo, path string) (bool, error) {
	return path == repo.Repository, nil
}
