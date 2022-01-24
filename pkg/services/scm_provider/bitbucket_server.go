package scm_provider

import (
	"context"
	"fmt"
	"strings"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
)

type BitbucketServerProvider struct {
	client      *bitbucketv1.APIClient
	projectKey  string
	allBranches bool
}

var _ SCMProviderService = &BitbucketServerProvider{}

func NewBitbucketServerProviderBasicAuth(ctx context.Context, username, password, url, projectKey string, allBranches bool) (*BitbucketServerProvider, error) {
	bitbucketConfig := bitbucketv1.NewConfiguration(url)
	// Avoid the XSRF check
	bitbucketConfig.AddDefaultHeader("x-atlassian-token", "no-check")
	bitbucketConfig.AddDefaultHeader("x-requested-with", "XMLHttpRequest")

	ctx = context.WithValue(ctx, bitbucketv1.ContextBasicAuth, bitbucketv1.BasicAuth{
		UserName: username,
		Password: password,
	})
	return newBitbucketServerProvider(ctx, bitbucketConfig, projectKey, allBranches)
}

func NewBitbucketServerProviderNoAuth(ctx context.Context, url, projectKey string, allBranches bool) (*BitbucketServerProvider, error) {
	return newBitbucketServerProvider(ctx, bitbucketv1.NewConfiguration(url), projectKey, allBranches)
}

func newBitbucketServerProvider(ctx context.Context, bitbucketConfig *bitbucketv1.Configuration, projectKey string, allBranches bool) (*BitbucketServerProvider, error) {
	if !strings.HasSuffix(bitbucketConfig.BasePath, "/rest") {
		bitbucketConfig.BasePath = bitbucketConfig.BasePath + "/rest"
	}
	bitbucketClient := bitbucketv1.NewAPIClient(ctx, bitbucketConfig)

	return &BitbucketServerProvider{
		client:      bitbucketClient,
		projectKey:  projectKey,
		allBranches: allBranches,
	}, nil
}

func (b *BitbucketServerProvider) ListRepos(_ context.Context, cloneProtocol string) ([]*Repository, error) {
	paged := map[string]interface{}{
		"limit": 100,
	}
	repos := []*Repository{}
	for {
		response, err := b.client.DefaultApi.GetRepositoriesWithOptions(b.projectKey, paged)
		if err != nil {
			return nil, fmt.Errorf("error listing repositories for %s: %v", b.projectKey, err)
		}
		repositories, err := bitbucketv1.GetRepositoriesResponse(response)
		if err != nil {
			return nil, fmt.Errorf("error parsing repositories response %s: %v", response.Values, err)
		}
		for _, bitbucketRepo := range repositories {
			var url string
			switch cloneProtocol {
			// Default to SSH if unspecified (i.e. if "").
			case "", "ssh":
				url = getCloneURLFromLinks(bitbucketRepo.Links.Clone, "ssh")
			case "https":
				url = getCloneURLFromLinks(bitbucketRepo.Links.Clone, "http")
			default:
				return nil, fmt.Errorf("unknown clone protocol for Bitbucket Server %v", cloneProtocol)
			}

			branches, err := b.listBranches(&bitbucketRepo)
			if err != nil {
				return nil, fmt.Errorf("error listing branches for %s/%s: %v", b.projectKey, bitbucketRepo.Name, err)
			}

			for _, branch := range branches {
				repos = append(repos, &Repository{
					Organization: bitbucketRepo.Project.Key,
					Repository:   bitbucketRepo.Name,
					URL:          url,
					Branch:       branch.DisplayID,
					SHA:          branch.LatestCommit,
					Labels:       []string{},
				})
			}
		}
		hasNextPage, nextPageStart := bitbucketv1.HasNextPage(response)
		if !hasNextPage {
			break
		}
		paged["start"] = nextPageStart
	}
	return repos, nil
}

func (b *BitbucketServerProvider) RepoHasPath(_ context.Context, repo *Repository, path string) (bool, error) {
	opts := map[string]interface{}{
		"limit": 100,
		"at":    repo.Branch,
	}
	// No need to query for all pages here
	response, err := b.client.DefaultApi.StreamFiles_42(repo.Organization, repo.Repository, path, opts)
	if response != nil && response.StatusCode == 404 {
		// The path requested does not exist at the supplied commit.
		return false, nil
	}
	if response != nil && response.StatusCode == 400 {
		// If the path is a file, the first call will return 400: The path requested is not a directory at the supplied commit.
		// Simply retry with an API call that works with files and expect a 200 return code
		opts["type_"] = true // Only request the type, we don't need the content
		response, err := b.client.DefaultApi.GetContent_0(repo.Organization, repo.Repository, path, opts)
		if response != nil && response.StatusCode == 404 {
			// File not found
			return false, nil
		}
		if err != nil {
			return false, err
		}
		// 200 ok
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (b *BitbucketServerProvider) listBranches(repo *bitbucketv1.Repository) ([]bitbucketv1.Branch, error) {
	// If we don't specifically want to query for all branches, just use the default branch and call it a day.
	if !b.allBranches {
		response, err := b.client.DefaultApi.GetDefaultBranch(repo.Project.Key, repo.Name)
		if err != nil {
			return nil, err
		}
		branch, err := bitbucketv1.GetBranchResponse(response)
		if err != nil {
			return nil, err
		}
		return []bitbucketv1.Branch{branch}, nil
	}
	// Otherwise, scrape the GetBranches API.
	branches := []bitbucketv1.Branch{}
	paged := map[string]interface{}{
		"limit": 100,
	}
	for {
		response, err := b.client.DefaultApi.GetBranches(repo.Project.Key, repo.Name, paged)
		if err != nil {
			return nil, fmt.Errorf("error listing branches for %s/%s: %v", repo.Project.Key, repo.Name, err)
		}
		bitbucketBranches, err := bitbucketv1.GetBranchesResponse(response)
		if err != nil {
			return nil, fmt.Errorf("error parsing branches response %s: %v", response.Values, err)
		}

		branches = append(branches, bitbucketBranches...)

		hasNextPage, nextPageStart := bitbucketv1.HasNextPage(response)
		if !hasNextPage {
			break
		}
		paged["start"] = nextPageStart
	}
	return branches, nil
}

func getCloneURLFromLinks(links []bitbucketv1.CloneLink, name string) string {
	for _, link := range links {
		if link.Name == name {
			return link.Href
		}
	}
	return ""
}
