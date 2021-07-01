package scm_provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// func checkRateLimit(t *testing.T, err error) {
// 	// Check if we've hit a rate limit, don't fail the test if so.
// 	if err != nil && strings.Contains(err.Error(), "rate limit exceeded") {
// 		allowRateLimitErrors := os.Getenv("CI") == ""
// 		t.Logf("Got a rate limit error, consider setting $Gitlab_TOKEN to increase your Gitlab API rate limit: %v\n", err)
// 		if allowRateLimitErrors {
// 			t.SkipNow()
// 		} else {
// 			t.FailNow()
// 		}
// 	}
// }

func TestGitlabListRepos(t *testing.T) {
	cases := []struct {
		name, proto, url      string
		hasError, allBranches bool
		branches              []string
	}{
		{
			name:     "blank protocol",
			url:      "git@gitlab.com:test-argocd-proton/argocd.git",
			branches: []string{"master"},
		},
		{
			name:  "ssh protocol",
			proto: "ssh",
			url:   "git@gitlab.com:test-argocd-proton/argocd.git",
		},
		{
			name:  "https protocol",
			proto: "https",
			url:   "https://gitlab.com/test-argocd-proton/argocd.git",
		},
		{
			name:     "other protocol",
			proto:    "other",
			hasError: true,
		},
		{
			name:        "all branches",
			allBranches: true,
			url:         "git@gitlab.com:test-argocd-proton/argocd.git",
			branches:    []string{"master", "pipeline-1310077506"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			provider, _ := NewGitlabProvider(context.Background(), "test-argocd-proton", "", "", c.allBranches)
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
					if r.Repository == "test-argocd-proton/argocd" {
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

func TestGitlabHasPath(t *testing.T) {
	host, _ := NewGitlabProvider(context.Background(), "test-argocd-proton", "", "", false)
	repo := &Repository{
		Organization: "test-argocd-proton",
		Repository:   "test-argocd-proton/argocd",
		Branch:       "master",
	}
	ok, err := host.RepoHasPath(context.Background(), repo, "argocd/")
	assert.Nil(t, err)
	assert.True(t, ok)

	ok, err = host.RepoHasPath(context.Background(), repo, "notathing/")
	assert.Nil(t, err)
	assert.False(t, ok)
}
