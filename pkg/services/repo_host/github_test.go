package repo_host

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGithubListRepos(t *testing.T) {
	cases := []struct {
		name, proto, url string
		hasError         bool
	}{
		{
			name: "blank protocol",
			url:  "git@github.com:argoproj-labs/applicationset.git",
		},
		{
			name:  "ssh protocol",
			proto: "ssh",
			url:   "git@github.com:argoproj-labs/applicationset.git",
		},
		{
			name:  "https protocol",
			proto: "https",
			url:   "https://github.com/argoproj-labs/applicationset.git",
		},
		{
			name:     "other protocol",
			proto:    "other",
			hasError: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			host, _ := NewGithubRepoHost(context.Background(), "argoproj-labs", "", "", false)
			repos, err := host.ListRepos(context.Background(), c.proto)
			if c.hasError {
				assert.NotNil(t, err)
			} else {
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
				assert.Equal(t, c.url, repo.URL)
			}
		})
	}
}

func TestGithubHasPath(t *testing.T) {
	host, _ := NewGithubRepoHost(context.Background(), "argoproj-labs", "", "", false)
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
