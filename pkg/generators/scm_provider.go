package generators

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argoprojiov1alpha1 "github.com/argoproj/applicationset/api/v1alpha1"
	"github.com/argoproj/applicationset/pkg/services/scm_provider"
)

var _ Generator = (*SCMProviderGenerator)(nil)

const (
	DefaultSCMProviderRequeueAfterSeconds = 30 * time.Minute
)

type SCMProviderGenerator struct {
	client client.Client
	// Testing hooks.
	overrideProvider scm_provider.SCMProviderService
}

func NewSCMProviderGenerator(client client.Client) Generator {
	return &SCMProviderGenerator{client: client}
}

func (g *SCMProviderGenerator) GetRequeueAfter(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) time.Duration {
	// Return a requeue default of 30 minutes, if no default is specified.

	if appSetGenerator.SCMProvider.RequeueAfterSeconds != nil {
		return time.Duration(*appSetGenerator.SCMProvider.RequeueAfterSeconds) * time.Second
	}

	return DefaultSCMProviderRequeueAfterSeconds
}

func (g *SCMProviderGenerator) GetTemplate(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator) *argoprojiov1alpha1.ApplicationSetTemplate {
	return &appSetGenerator.SCMProvider.Template
}

func (g *SCMProviderGenerator) GenerateParams(appSetGenerator *argoprojiov1alpha1.ApplicationSetGenerator, applicationSetInfo *argoprojiov1alpha1.ApplicationSet) ([]map[string]string, error) {
	if appSetGenerator == nil {
		return nil, EmptyAppSetGeneratorError
	}

	if appSetGenerator.SCMProvider == nil {
		return nil, EmptyAppSetGeneratorError
	}

	ctx := context.Background()

	// Create the SCM provider helper.
	providerConfig := appSetGenerator.SCMProvider
	var provider scm_provider.SCMProviderService
	if g.overrideProvider != nil {
		provider = g.overrideProvider
	} else if providerConfig.Github != nil {
		token, err := g.getSecretRef(ctx, providerConfig.Github.TokenRef, applicationSetInfo.Namespace)
		if err != nil {
			return nil, fmt.Errorf("error fetching Github token: %v", err)
		}
		provider, err = scm_provider.NewGithubProvider(ctx, providerConfig.Github.Organization, token, providerConfig.Github.API, providerConfig.Github.AllBranches, providerConfig.Github.AllPullRequests)
		if err != nil {
			return nil, fmt.Errorf("error initializing Github service: %v", err)
		}
	} else if providerConfig.Gitlab != nil {
		token, err := g.getSecretRef(ctx, providerConfig.Gitlab.TokenRef, applicationSetInfo.Namespace)
		if err != nil {
			return nil, fmt.Errorf("error fetching Gitlab token: %v", err)
		}
		provider, err = scm_provider.NewGitlabProvider(ctx, providerConfig.Gitlab.Group, token, providerConfig.Gitlab.API, providerConfig.Gitlab.AllBranches, providerConfig.Gitlab.IncludeSubgroups, providerConfig.Gitlab.AllPullRequests)
		if err != nil {
			return nil, fmt.Errorf("error initializing Gitlab service: %v", err)
		}
	} else {
		return nil, fmt.Errorf("no SCM provider implementation configured")
	}

	// Find all the available repos.
	repos, err := scm_provider.ListRepos(ctx, provider, providerConfig.Filters, providerConfig.CloneProtocol)
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
			"sha":          repo.SHA,
			"labels":       strings.Join(repo.Labels, ","),
		})
	}
	return params, nil
}

func (g *SCMProviderGenerator) getSecretRef(ctx context.Context, ref *argoprojiov1alpha1.SecretRef, namespace string) (string, error) {
	if ref == nil {
		return "", nil
	}

	secret := &corev1.Secret{}
	err := g.client.Get(
		ctx,
		client.ObjectKey{
			Name:      ref.SecretName,
			Namespace: namespace,
		},
		secret)
	if err != nil {
		return "", fmt.Errorf("error fetching secret %s/%s: %v", namespace, ref.SecretName, err)
	}
	tokenBytes, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("key %q in secret %s/%s not found", ref.Key, namespace, ref.SecretName)
	}
	return string(tokenBytes), nil
}
