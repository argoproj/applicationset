package scm_provider

import (
	"context"
	"fmt"
	"os"

	gitlab "github.com/xanzy/go-gitlab"
)

type GitlabProvider struct {
	client           *gitlab.Client
	organization     string
	allBranches      bool
	includeSubgroups bool
}

var _ SCMProviderService = &GitlabProvider{}

func NewGitlabProvider(ctx context.Context, organization string, token string, url string, allBranches, includeSubgroups bool) (*GitlabProvider, error) {
	// Undocumented environment variable to set a default token, to be used in testing to dodge anonymous rate limits.
	if token == "" {
		token = os.Getenv("GITLAB_TOKEN")
	}
	var client *gitlab.Client
	if url == "" {
		var err error
		client, err = gitlab.NewClient(token)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		client, err = gitlab.NewClient(token, gitlab.WithBaseURL(url))
		if err != nil {
			return nil, err
		}
	}
	return &GitlabProvider{client: client, organization: organization, allBranches: allBranches, includeSubgroups: includeSubgroups}, nil
}

func (g *GitlabProvider) ListRepos(ctx context.Context, cloneProtocol string) ([]*Repository, error) {
	opt := &gitlab.ListGroupProjectsOptions{
		ListOptions:      gitlab.ListOptions{PerPage: 100},
		IncludeSubgroups: &g.includeSubgroups,
	}
	repos := []*Repository{}
	for {
		gitlabRepos, resp, err := g.client.Groups.ListGroupProjects(g.organization, opt)
		if err != nil {
			return nil, fmt.Errorf("error listing projects for %s: %v", g.organization, err)
		}
		for _, gitlabRepo := range gitlabRepos {
			var url string
			switch cloneProtocol {
			// Default to SSH if unspecified (i.e. if "").
			case "", "ssh":
				url = gitlabRepo.SSHURLToRepo
			case "https":
				url = gitlabRepo.HTTPURLToRepo
			default:
				return nil, fmt.Errorf("unknown clone protocol for Gitlab %v", cloneProtocol)
			}

			branches, err := g.listBranches(ctx, gitlabRepo)
			if err != nil {
				return nil, fmt.Errorf("error listing branches for %s/%s: %v", g.organization, gitlabRepo.Name, err)
			}

			for _, branch := range branches {
				repos = append(repos, &Repository{
					Organization: gitlabRepo.Namespace.FullPath,
					Repository:   gitlabRepo.Path,
					URL:          url,
					Branch:       branch,
					Labels:       gitlabRepo.TagList,
				})
			}
		}
		if resp.CurrentPage >= resp.TotalPages {
			break
		}
		opt.Page = resp.NextPage
	}
	return repos, nil
}

func (g *GitlabProvider) RepoHasPath(_ context.Context, repo *Repository, path string) (bool, error) {
	p, _, err := g.client.Projects.GetProject(repo.Organization+"/"+repo.Repository, nil)
	if err != nil {
		return false, err
	}
	_, resp, err := g.client.Repositories.ListTree(p.ID, &gitlab.ListTreeOptions{
		Path: &path,
		Ref:  &repo.Branch,
	})
	if resp.TotalItems == 0 {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (g *GitlabProvider) listBranches(_ context.Context, repo *gitlab.Project) ([]string, error) {
	// If we don't specifically want to query for all branches, just use the default branch and call it a day.
	if !g.allBranches {
		return []string{repo.DefaultBranch}, nil
	}
	// Otherwise, scrape the ListBranches API.
	opt := &gitlab.ListBranchesOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}
	branches := []string{}
	for {
		gitlabBranches, resp, err := g.client.Branches.ListBranches(repo.ID, opt)
		if err != nil {
			return nil, err
		}
		for _, gitlabBranch := range gitlabBranches {
			branches = append(branches, gitlabBranch.Name)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return branches, nil
}
