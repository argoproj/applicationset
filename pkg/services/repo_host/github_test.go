package repo_host

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGithubListRepos(t *testing.T) {
	cases := []struct {
		name, proto, url      string
		hasError, allBranches bool
		branches              []string
	}{
		{
			name:     "blank protocol",
			url:      "git@github.com:argoproj-labs/applicationset.git",
			branches: []string{"master"},
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
		{
			name:        "all branches",
			allBranches: true,
			url:         "git@github.com:argoproj-labs/applicationset.git",
			branches:    []string{"master", "release-0.1.0"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			host, _ := NewGithubRepoHost(context.Background(), "argoproj-labs", "", "", c.allBranches)
			rawRepos, err := host.ListRepos(context.Background(), c.proto)
			if c.hasError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				// Just check that this one project shows up. Not a great test but better thing nothing?
				repos := []*HostedRepo{}
				branches := []string{}
				for _, r := range rawRepos {
					if r.Repository == "applicationset" {
						repos = append(repos, r)
						branches = append(branches, r.Branch)
					}
				}
				assert.NotEmpty(t, repos)
				assert.Equal(t, c.url, repos[0].URL)
				for _, b := range c.branches {
					assert.Contains(t, branches, b)
				}
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
