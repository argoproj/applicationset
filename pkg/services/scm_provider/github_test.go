package scm_provider

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func checkRateLimit(t *testing.T, err error) {
	// Check if we've hit a rate limit, don't fail the test if so.
	if err != nil && (strings.Contains(err.Error(), "rate limit exceeded") ||
		(strings.Contains(err.Error(), "API rate limit") && strings.Contains(err.Error(), "still exceeded"))) {

		allowRateLimitErrors := os.Getenv("CI") == ""
		t.Logf("Got a rate limit error, consider setting $GITHUB_TOKEN to increase your GitHub API rate limit: %v\n", err)
		if allowRateLimitErrors {
			t.SkipNow()
		} else {
			t.FailNow()
		}
	}
}

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
			provider, _ := NewGithubProvider(context.Background(), "argoproj-labs", "", "", c.allBranches)
			rawRepos, err := provider.ListRepos(context.Background(), c.proto)
			if c.hasError {
				assert.NotNil(t, err)
			} else {
				checkRateLimit(t, err)
				assert.Nil(t, err)
				// Just check that this one project shows up. Not a great test but better thing nothing?
				repos := []*Repository{}
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
	host, _ := NewGithubProvider(context.Background(), "argoproj-labs", "", "", false)
	repo := &Repository{
		Organization: "argoproj-labs",
		Repository:   "applicationset",
		Branch:       "master",
	}
	ok, err := host.RepoHasPath(context.Background(), repo, "pkg/")
	checkRateLimit(t, err)
	assert.Nil(t, err)
	assert.True(t, ok)

	ok, err = host.RepoHasPath(context.Background(), repo, "notathing/")
	checkRateLimit(t, err)
	assert.Nil(t, err)
	assert.False(t, ok)
}
