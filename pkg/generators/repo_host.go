package generators

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/services/repo_host"
)

var _ Generator = (*RepoHostGenerator)(nil)

const (
	DefaultRepoHostRequeueAfterSeconds = 30 * time.Minute
)

type RepoHostGenerator struct {
	client client.Client
	// Testing hooks.
	overrideHost repo_host.RepoHostService
}

func NewRepoHostGenerator(client client.Client) Generator {
	return &RepoHostGenerator{client: client}
}

func (g *RepoHostGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	// Return a requeue default of 30 minutes, if no default is specified.

	if appSetGenerator.RepoHost.RequeueAfterSeconds != nil {
		return time.Duration(*appSetGenerator.RepoHost.RequeueAfterSeconds) * time.Second
	}

	return DefaultRepoHostRequeueAfterSeconds
}

func (g *RepoHostGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return &appSetGenerator.RepoHost.Template
}

func (g *RepoHostGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator, applicationSetInfo *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {
	if appSetGenerator == nil {
		return nil, EmptyAppSetGeneratorError
	}

	if appSetGenerator.RepoHost == nil {
		return nil, EmptyAppSetGeneratorError
	}

	ctx := context.Background()

	// Create the repo host.
	hostConfig := appSetGenerator.RepoHost
	var host repo_host.RepoHostService
	if g.overrideHost != nil {
		host = g.overrideHost
	} else if hostConfig.Github != nil {
		token, err := g.getSecretRef(ctx, hostConfig.Github.TokenRef, applicationSetInfo.Namespace)
		if err != nil {
			return nil, fmt.Errorf("error fetching Github token: %v", err)
		}
		host, err = repo_host.NewGithubRepoHost(ctx, hostConfig.Github.Organization, token, hostConfig.Github.API)
		if err != nil {
			return nil, fmt.Errorf("error initializing Github service: %v", err)
		}
	} else {
		return nil, fmt.Errorf("no repository host provider configured")
	}

	// Find all the available repos.
	repos, err := repo_host.ListRepos(ctx, host, hostConfig.Filters)
	if err != nil {
		return nil, fmt.Errorf("error listing repos: %v", err)
	}
	params := make([]map[string]string, 0, len(repos))
	for _, repo := range repos {
		params = append(params, map[string]string{
			"organization": repo.Organization,
			"repository":   repo.Repository,
			"url":          repo.URL,
			"branch":       repo.Branch,
			"labels":       strings.Join(repo.Labels, ","),
		})
	}
	return params, nil
}

func (g *RepoHostGenerator) getSecretRef(ctx context.Context, ref *argoprojiov1alpha1.SecretRef, namespace string) (string, error) {
	if ref == nil {
		return "", nil
	}

	secret := &corev1.Secret{}
	err := g.client.Get(
		ctx,
		client.ObjectKey{
			Name:      ref.Name,
			Namespace: namespace,
		},
		secret)
	if err != nil {
		return "", fmt.Errorf("error fetching secret %s/%s: %v", namespace, ref.Name, err)
	}
	tokenBytes, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("key %q in secret %s/%s not found", ref.Key, namespace, ref.Name)
	}
	return string(tokenBytes), nil
}
