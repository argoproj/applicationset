package pull_request

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	bitbucketv1 "github.com/gfleury/go-bitbucket-v1"
)

type BitbucketService struct {
	client         *bitbucketv1.APIClient
	projectKey     string
	repositorySlug string
	branchMatch    *regexp.Regexp
	// Not supported for PRs by Bitbucket Server
	// labels         []string
}

var _ PullRequestService = (*BitbucketService)(nil)

func NewBitbucketServiceBasicAuth(ctx context.Context, username, password, url, projectKey, repositorySlug string, branchMatch *string) (PullRequestService, error) {
	bitbucketConfig := bitbucketv1.NewConfiguration(url)
	// Avoid the XSRF check
	bitbucketConfig.AddDefaultHeader("x-atlassian-token", "no-check")
	bitbucketConfig.AddDefaultHeader("x-requested-with", "XMLHttpRequest")

	ctx = context.WithValue(ctx, bitbucketv1.ContextBasicAuth, bitbucketv1.BasicAuth{
		UserName: username,
		Password: password,
	})
	return newBitbucketService(ctx, bitbucketConfig, projectKey, repositorySlug, branchMatch)
}

func NewBitbucketServiceNoAuth(ctx context.Context, url, projectKey, repositorySlug string, branchMatch *string) (PullRequestService, error) {
	return newBitbucketService(ctx, bitbucketv1.NewConfiguration(url), projectKey, repositorySlug, branchMatch)
}

func newBitbucketService(ctx context.Context, bitbucketConfig *bitbucketv1.Configuration, projectKey, repositorySlug string, branchMatch *string) (PullRequestService, error) {
	if !strings.HasSuffix(bitbucketConfig.BasePath, "/rest") {
		bitbucketConfig.BasePath = bitbucketConfig.BasePath + "/rest"
	}
	bitbucketClient := bitbucketv1.NewAPIClient(ctx, bitbucketConfig)

	var branchMatchRegexp *regexp.Regexp
	if branchMatch != nil {
		var err error
		branchMatchRegexp, err = regexp.Compile(*branchMatch)
		if err != nil {
			return nil, fmt.Errorf("error compiling BranchMatch regexp %q: %v", *branchMatch, err)
		}
	}

	return &BitbucketService{
		client:         bitbucketClient,
		projectKey:     projectKey,
		repositorySlug: repositorySlug,
		branchMatch:    branchMatchRegexp,
	}, nil
}

func (b *BitbucketService) List(_ context.Context) ([]*PullRequest, error) {
	paged := map[string]interface{}{
		"limit": 100,
	}

	pullRequests := []*PullRequest{}
	for {
		response, err := b.client.DefaultApi.GetPullRequestsPage(b.projectKey, b.repositorySlug, paged)
		if err != nil {
			return nil, fmt.Errorf("error listing pull requests for %s/%s: %v", b.projectKey, b.repositorySlug, err)
		}
		pulls, err := bitbucketv1.GetPullRequestsResponse(response)
		if err != nil {
			return nil, fmt.Errorf("error parsing pull request response %s: %v", response.Values, err)
		}

		for _, pull := range pulls {
			if b.branchMatch != nil && !b.branchMatch.MatchString(pull.FromRef.DisplayID) {
				continue
			}
			pullRequests = append(pullRequests, &PullRequest{
				Number:  pull.ID,
				Branch:  pull.FromRef.DisplayID,    // ID: refs/heads/main DisplayID: main
				HeadSHA: pull.FromRef.LatestCommit, // This is not defined in the official docs, but works in practice
			})
		}

		hasNextPage, nextPageStart := bitbucketv1.HasNextPage(response)
		if !hasNextPage {
			break
		}
		paged["start"] = nextPageStart
	}
	return pullRequests, nil
}
