package scm_provider

import (
	"context"
	"fmt"
	"os"

	gitlab "github.com/xanzy/go-gitlab"
)

type GitlabProvider struct {
	client       *gitlab.Client
	group string
	allBranches  bool
}

var _ SCMProviderService = &GitlabProvider{}

func newTrue() *bool {
	b := true
	return &b
}

func NewGitlabProvider(ctx context.Context, organization string, token string, url string, allBranches bool) (*GitlabProvider, error) {
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
	return &GitlabProvider{client: client, organization: organization, allBranches: allBranches}, nil
}

func (g *GitlabProvider) ListRepos(ctx context.Context, cloneProtocol string) ([]*Repository, error) {
	opt := &gitlab.ListGroupProjectsOptions{
		ListOptions:      gitlab.ListOptions{PerPage: 100},
		IncludeSubgroups: newTrue(),
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
					Organization: g.organization,
					Repository:   gitlabRepo.PathWithNamespace,
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

func (g *GitlabProvider) RepoHasPath(ctx context.Context, repo *Repository, path string) (bool, error) {
	p, _, err := g.client.Projects.GetProject(repo.Repository, nil)
	if err != nil {
		return false, fmt.Errorf("Error retrieving project %s", repo.Repository)
	}
	_, resp, err := g.client.RepositoryFiles.GetFileMetaData(p.ID, path, &gitlab.GetFileMetaDataOptions{
		Ref: &repo.Branch,
	})
	// 404s are not an error here, just a normal false.
	if resp != nil && resp.StatusCode == 404 {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (g *GitlabProvider) listBranches(ctx context.Context, repo *gitlab.Project) ([]string, error) {
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
