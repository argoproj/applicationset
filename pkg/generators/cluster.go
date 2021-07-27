package generators

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	argoappv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v2/util/settings"

	argoappsetv1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/utils"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ArgoCDSecretTypeLabel   = "argocd.argoproj.io/secret-type"
	ArgoCDSecretTypeCluster = "cluster"
)

var _ Generator = (*ClusterGenerator)(nil)

// ClusterGenerator generates Applications for some or all clusters registered with ArgoCD.
type ClusterGenerator struct {
	client.Client
	ctx       context.Context
	clientset kubernetes.Interface
	// namespace is the Argo CD namespace
	namespace       string
	settingsManager *settings.SettingsManager
}

func NewClusterGenerator(c client.Client, ctx context.Context, clientset kubernetes.Interface, namespace string) Generator {

	settingsManager := settings.NewSettingsManager(ctx, clientset, namespace)

	g := &ClusterGenerator{
		Client:          c,
		ctx:             ctx,
		clientset:       clientset,
		namespace:       namespace,
		settingsManager: settingsManager,
	}
	return g
}

func (g *ClusterGenerator) GetRequeueAfter(appSetGenerator *argoappsetv1alpha1.ApplicationSetGenerator) time.Duration {
	return NoRequeueAfter
}

func (g *ClusterGenerator) GetTemplate(appSetGenerator *argoappsetv1alpha1.ApplicationSetGenerator) *argoappsetv1alpha1.ApplicationSetTemplate {
	return &appSetGenerator.Clusters.Template
}

func (g *ClusterGenerator) GenerateParams(
	appSetGenerator *argoappsetv1alpha1.ApplicationSetGenerator, _ *argoappsetv1alpha1.ApplicationSet) ([]map[string]string, error) {

	if appSetGenerator == nil {
		return nil, EmptyAppSetGeneratorError
	}

	if appSetGenerator.Clusters == nil {
		return nil, nil
	}

	paramsSet, err := g.clusterParams(appSetGenerator.Clusters)
	if err != nil {
		return nil, err
	}

	// Finally, enrich all cluster params with the common values.
	for _, params := range paramsSet {
		for key, value := range appSetGenerator.Clusters.Values {
			params[fmt.Sprintf("values.%s", key)] = value
		}
	}

	return paramsSet, nil
}

func (g *ClusterGenerator) clusterParams(config *argoappsetv1alpha1.ClusterGenerator) ([]map[string]string, error) {
	// If no selector was specified, this will yield all cluster secrets. The
	// local cluster will be included if and only if it has a secret.
	clusterSecrets, err := g.findClusterSecrets(&config.Selector)
	if err != nil {
		return nil, err
	}

	paramsSet := []map[string]string{}

	// If a selector was specified, we generate params for any matching cluster,
	// which may include the local cluster if it has a secret.
	if len(config.Selector.MatchExpressions) > 0 ||
		len(config.Selector.MatchLabels) > 0 {
		for _, secret := range clusterSecrets {
			paramsSet = append(paramsSet, paramsForSecret(&secret))
		}
		return paramsSet, nil
	}

	// No selector, so if a list of cluster names was provided, use that.
	if len(config.Names) > 0 {
		for _, name := range config.Names {
			secret, ok := clusterSecrets[name]
			if !ok {
				log.WithField("cluster", name).Warn("skipping unknown cluster name")
				continue
			}
			paramsSet = append(paramsSet, paramsForSecret(&secret))
		}
		return paramsSet, nil
	}

	// Assume a wildcard selector.

	// ListClusters provides the name and server of every cluster including
	// local, but not labels or annotations - we need the secrets for those.
	clusters, err := utils.ListClusters(g.ctx, g.clientset, g.namespace)
	if err != nil {
		return nil, err
	}

	for _, cluster := range clusters.Items {
		if secret, ok := clusterSecrets[cluster.Name]; ok {
			// Remote clusters, and the local cluster if it has a secret.
			paramsSet = append(paramsSet, paramsForSecret(&secret))
		} else {
			// Local cluster, which evidently does not have a secret.
			paramsSet = append(paramsSet, paramsForCluster(&cluster))
		}
	}
	return paramsSet, nil
}

func paramsForSecret(secret *corev1.Secret) map[string]string {
	params := map[string]string{
		"name":   sanitizeName(string(secret.Data["name"])),
		"server": string(secret.Data["server"]),
	}
	for key, value := range secret.ObjectMeta.Annotations {
		params[fmt.Sprintf("metadata.annotations.%s", key)] = value
	}
	for key, value := range secret.ObjectMeta.Labels {
		params[fmt.Sprintf("metadata.labels.%s", key)] = value
	}
	return params
}

func paramsForCluster(cluster *argoappv1alpha1.Cluster) map[string]string {
	return map[string]string{
		"name":   cluster.Name,
		"server": cluster.Server,
	}
}

// sanitizeName returns the provided cluster name, modified if necessary to meet
// the following rules:
//   1. contains no more than 253 characters
//   2. contains only lowercase alphanumeric characters, '-' or '.'
//   3. starts and ends with an alphanumeric character
func sanitizeName(name string) string {
	invalidDNSNameChars := regexp.MustCompile("[^-a-z0-9.]")
	maxDNSNameLength := 253

	name = strings.ToLower(name)
	name = invalidDNSNameChars.ReplaceAllString(name, "-")
	if len(name) > maxDNSNameLength {
		name = name[:maxDNSNameLength]
	}

	return strings.Trim(name, "-.")
}

// findClusterSecrets returns all cluster secret objects matching the provided
// filter, keyed by cluster name.
func (g *ClusterGenerator) findClusterSecrets(selector *metav1.LabelSelector) (map[string]corev1.Secret, error) {
	withCluster := metav1.CloneSelectorAndAddLabel(selector, ArgoCDSecretTypeLabel, ArgoCDSecretTypeCluster)
	secretSelector, err := metav1.LabelSelectorAsSelector(withCluster)
	if err != nil {
		return nil, err
	}

	clusterSecretList := &corev1.SecretList{}
	if err := g.Client.List(g.ctx, clusterSecretList, client.MatchingLabelsSelector{Selector: secretSelector}); err != nil {
		return nil, err
	}
	log.Debug("clusters matching labels", "count", len(clusterSecretList.Items))

	res := map[string]corev1.Secret{}
	for _, cluster := range clusterSecretList.Items {
		clusterName := string(cluster.Data["name"])
		res[clusterName] = cluster
	}

	return res, nil
}
