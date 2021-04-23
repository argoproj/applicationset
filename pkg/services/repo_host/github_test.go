package repo_host

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGithubListRepos(t *testing.T) {
	host, _ := NewGithubRepoHost(context.Background(), "argoproj-labs", "", "")
	repos, err := host.ListRepos(context.Background())
	assert.Nil(t, err)
	// Just check that this one project shows up. Not a great test but better thing nothing?
	var repo *HostedRepo
	for _, r := range repos {
		if r.Repository == "applicationset" {
			repo = r
			break
		}
	}
	assert.NotNil(t, repo)
	assert.Equal(t, "git@github.com:argoproj-labs/applicationset.git", repo.URL)
}

func TestGithubHasPath(t *testing.T) {
	host, _ := NewGithubRepoHost(context.Background(), "argoproj-labs", "", "")
	repo := &HostedRepo{
		Organization: "argoproj-labs",
		Repository:   "applicationset",
		Branch:       "master",
	}
	ok, err := host.RepoHasPath(context.Background(), repo, "pkg/")
	assert.Nil(t, err)
	assert.True(t, ok)

	ok, err = host.RepoHasPath(context.Background(), repo, "notathing/")
	assert.Nil(t, err)
	assert.False(t, ok)
}
