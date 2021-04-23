package repo_host

import (
	"context"

	"github.com/google/go-github/v35/github"
	"golang.org/x/oauth2"
)

type GithubRepoHost struct {
	client       *github.Client
	organization string
}

var _ RepoHostService = &GithubRepoHost{}

func NewGithubRepoHost(ctx context.Context, organization string, token string, url string) (*GithubRepoHost, error) {
	var ts oauth2.TokenSource
	if token != "" {
		ts = oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
	}
	httpClient := oauth2.NewClient(ctx, ts)
	var client *github.Client
	if url == "" {
		client = github.NewClient(httpClient)
	} else {
		var err error
		client, err = github.NewEnterpriseClient(url, url, httpClient)
		if err != nil {
			return nil, err
		}
	}
	return &GithubRepoHost{client: client, organization: organization}, nil
}

func (g *GithubRepoHost) ListRepos(ctx context.Context) ([]*HostedRepo, error) {
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	repos := []*HostedRepo{}
	for {
		githubRepos, resp, err := g.client.Repositories.ListByOrg(ctx, g.organization, opt)
		if err != nil {
			return nil, err
		}
		for _, githubRepo := range githubRepos {
			repos = append(repos, &HostedRepo{
				Organization: githubRepo.Owner.GetName(),
				Repository:   githubRepo.GetName(),
				URL:          githubRepo.GetSSHURL(), // TODO Config flag for CloneURL (i.e. https://)?
				Branch:       githubRepo.GetDefaultBranch(),
				Labels:       githubRepo.Topics,
			})
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return repos, nil
}

func (g *GithubRepoHost) RepoHasPath(ctx context.Context, repo *HostedRepo, path string) (bool, error) {
	_, _, resp, err := g.client.Repositories.GetContents(ctx, repo.Organization, repo.Repository, path, &github.RepositoryContentGetOptions{
		Ref: repo.Branch,
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
