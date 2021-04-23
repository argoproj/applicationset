package repo_host

import (
	"context"
	"testing"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func strp(s string) *string {
	return &s
}

func TestFilterRepoMatch(t *testing.T) {
	host := &MockRepoHost{
		Repos: []*HostedRepo{
			{
				Repository: "one",
			},
			{
				Repository: "two",
			},
			{
				Repository: "three",
			},
			{
				Repository: "four",
			},
		},
	}
	filters := []argoprojiov1alpha1.RepoHostGeneratorFilter{
		{
			RepositoryMatch: strp("n|hr"),
		},
	}
	repos, err := ListRepos(context.Background(), host, filters)
	assert.Nil(t, err)
	assert.Len(t, repos, 2)
	assert.Equal(t, "one", repos[0].Repository)
	assert.Equal(t, "three", repos[1].Repository)
}

func TestFilterLabelMatch(t *testing.T) {
	host := &MockRepoHost{
		Repos: []*HostedRepo{
			{
				Repository: "one",
				Labels:     []string{"prod-one", "prod-two", "staging"},
			},
			{
				Repository: "two",
				Labels:     []string{"prod-two"},
			},
			{
				Repository: "three",
				Labels:     []string{"staging"},
			},
		},
	}
	filters := []argoprojiov1alpha1.RepoHostGeneratorFilter{
		{
			LabelMatch: strp("^prod-.*$"),
		},
	}
	repos, err := ListRepos(context.Background(), host, filters)
	assert.Nil(t, err)
	assert.Len(t, repos, 2)
	assert.Equal(t, "one", repos[0].Repository)
	assert.Equal(t, "two", repos[1].Repository)
}

func TestFilterPatchExists(t *testing.T) {
	host := &MockRepoHost{
		Repos: []*HostedRepo{
			{
				Repository: "one",
			},
			{
				Repository: "two",
			},
			{
				Repository: "three",
			},
		},
	}
	filters := []argoprojiov1alpha1.RepoHostGeneratorFilter{
		{
			PathExists: strp("two"),
		},
	}
	repos, err := ListRepos(context.Background(), host, filters)
	assert.Nil(t, err)
	assert.Len(t, repos, 1)
	assert.Equal(t, "two", repos[0].Repository)
}

func TestFilterRepoMatchBadRegexp(t *testing.T) {
	host := &MockRepoHost{
		Repos: []*HostedRepo{
			{
				Repository: "one",
			},
		},
	}
	filters := []argoprojiov1alpha1.RepoHostGeneratorFilter{
		{
			RepositoryMatch: strp("("),
		},
	}
	_, err := ListRepos(context.Background(), host, filters)
	assert.NotNil(t, err)
}

func TestFilterLabelMatchBadRegexp(t *testing.T) {
	host := &MockRepoHost{
		Repos: []*HostedRepo{
			{
				Repository: "one",
			},
		},
	}
	filters := []argoprojiov1alpha1.RepoHostGeneratorFilter{
		{
			LabelMatch: strp("("),
		},
	}
	_, err := ListRepos(context.Background(), host, filters)
	assert.NotNil(t, err)
}
