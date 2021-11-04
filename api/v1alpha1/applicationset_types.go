/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/argoproj-labs/applicationset/common"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Utility struct for a reference to a secret key.
type SecretRef struct {
	SecretName string `json:"secretName"`
	Key        string `json:"key"`
}

// ApplicationSet is a set of Application resources
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=applicationsets,shortName=appset;appsets
type ApplicationSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ApplicationSetSpec   `json:"spec"`
	Status ApplicationSetStatus `json:"status,omitempty"`
}

// ApplicationSetSpec represents a class of application set state.
type ApplicationSetSpec struct {
	Generators []ApplicationSetGenerator `json:"generators"`
	Template   ApplicationSetTemplate    `json:"template"`
	SyncPolicy *ApplicationSetSyncPolicy `json:"syncPolicy,omitempty"`
}

// ApplicationSetSyncPolicy configures how generated Applications will relate to their
// ApplicationSet.
type ApplicationSetSyncPolicy struct {
	// PreserveResourcesOnDeletion will preserve resources on deletion. If PreserveResourcesOnDeletion is set to true, these Applications will not be deleted.
	PreserveResourcesOnDeletion bool `json:"preserveResourcesOnDeletion,omitempty"`
}

// ApplicationSetTemplate represents argocd ApplicationSpec
type ApplicationSetTemplate struct {
	ApplicationSetTemplateMeta `json:"metadata"`
	Spec                       v1alpha1.ApplicationSpec `json:"spec"`
}

// ApplicationSetTemplateMeta represents the Argo CD application fields that may
// be used for Applications generated from the ApplicationSet (based on metav1.ObjectMeta)
type ApplicationSetTemplateMeta struct {
	Name        string            `json:"name,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Finalizers  []string          `json:"finalizers,omitempty"`
}

// ApplicationSetGenerator represents a generator at the top level of an ApplicationSet.
type ApplicationSetGenerator struct {
	List                    *ListGenerator        `json:"list,omitempty"`
	Clusters                *ClusterGenerator     `json:"clusters,omitempty"`
	Git                     *GitGenerator         `json:"git,omitempty"`
	Matrix                  *MatrixGenerator      `json:"matrix,omitempty"`
	Union                   *UnionGenerator       `json:"union,omitempty"`
	SCMProvider             *SCMProviderGenerator `json:"scmProvider,omitempty"`
	ClusterDecisionResource *DuckTypeGenerator    `json:"clusterDecisionResource,omitempty"`
	PullRequest             *PullRequestGenerator `json:"pullRequest,omitempty"`
}

// ApplicationSetNestedGenerator represents a generator nested within a combination-type generator (MatrixGenerator or
// UnionGenerator).
type ApplicationSetNestedGenerator struct {
	List                    *ListGenerator         `json:"list,omitempty"`
	Clusters                *ClusterGenerator      `json:"clusters,omitempty"`
	Git                     *GitGenerator          `json:"git,omitempty"`
	Matrix                  *NestedMatrixGenerator `json:"matrix,omitempty"`
	Union                   *NestedUnionGenerator  `json:"union,omitempty"`
	SCMProvider             *SCMProviderGenerator  `json:"scmProvider,omitempty"`
	ClusterDecisionResource *DuckTypeGenerator     `json:"clusterDecisionResource,omitempty"`
	PullRequest             *PullRequestGenerator  `json:"pullRequest,omitempty"`
}

type ApplicationSetNestedGenerators []ApplicationSetNestedGenerator

// ApplicationSetTerminalGenerator represents a generator nested within a nested generator (for example, a list within
// a union within a matrix). A generator at this level may not be a combination-type generator (MatrixGenerator or
// UnionGenerator). ApplicationSet enforces this nesting depth limit because CRDs do not support recursive types.
// https://github.com/kubernetes-sigs/controller-tools/issues/477
type ApplicationSetTerminalGenerator struct {
	List                    *ListGenerator        `json:"list,omitempty"`
	Clusters                *ClusterGenerator     `json:"clusters,omitempty"`
	Git                     *GitGenerator         `json:"git,omitempty"`
	SCMProvider             *SCMProviderGenerator `json:"scmProvider,omitempty"`
	ClusterDecisionResource *DuckTypeGenerator    `json:"clusterDecisionResource,omitempty"`
	PullRequest             *PullRequestGenerator `json:"pullRequest,omitempty"`
}

type ApplicationSetTerminalGenerators []ApplicationSetTerminalGenerator

// toApplicationSetNestedGenerators converts a terminal generator (a generator which cannot be a combination-type
// generator) to a "nested" generator. The conversion is for convenience, allowing generator g to be used where a nested
// generator is expected.
func (g ApplicationSetTerminalGenerators) toApplicationSetNestedGenerators() []ApplicationSetNestedGenerator {
	nestedGenerators := make([]ApplicationSetNestedGenerator, len(g))
	for i, terminalGenerator := range g {
		nestedGenerators[i] = ApplicationSetNestedGenerator{
			List:                    terminalGenerator.List,
			Clusters:                terminalGenerator.Clusters,
			Git:                     terminalGenerator.Git,
			SCMProvider:             terminalGenerator.SCMProvider,
			ClusterDecisionResource: terminalGenerator.ClusterDecisionResource,
			PullRequest:             terminalGenerator.PullRequest,
		}
	}
	return nestedGenerators
}

// ListGenerator include items info
type ListGenerator struct {
	Elements []apiextensionsv1.JSON `json:"elements"`
	Template ApplicationSetTemplate `json:"template,omitempty"`
}

// MatrixGenerator generates the cartesian product of two sets of parameters. The parameters are defined by two nested
// generators.
type MatrixGenerator struct {
	Generators []ApplicationSetNestedGenerator `json:"generators"`
	Template   ApplicationSetTemplate          `json:"template,omitempty"`
}

// NestedMatrixGenerator is a MatrixGenerator nested under another combination-type generator (MatrixGenerator or
// UnionGenerator). NestedMatrixGenerator does not have an override template, because template overriding has no meaning
// within the constituent generators of combination-type generators.
type NestedMatrixGenerator struct {
	Generators ApplicationSetTerminalGenerators `json:"generators"`
}

// ToMatrixGenerator converts a NestedMatrixGenerator to a MatrixGenerator. This conversion is for convenience, allowing
// a NestedMatrixGenerator to be used where a MatrixGenerator is expected (of course, the converted generator will have
// no override template).
func (g NestedMatrixGenerator) ToMatrixGenerator() *MatrixGenerator {
	return &MatrixGenerator{
		Generators: g.Generators.toApplicationSetNestedGenerators(),
	}
}

// UnionGenerator takes the union of two or more generators. Where the values for all specified merge keys are equal
// between two sets of generated parameters, the parameter sets will be merged with the parameters from the latter
// generator taking precedence.
// For example, if the first generator produced [{a: '1', b: '2'}, {c: '1', d: '1'}] and the second generator produced
// [{'a': 'override'}], the united parameters would be [{a: 'override', b: '1'}, {c: '1', d: '1'}].
//
// UnionGenerator supports template overriding. If a UnionGenerator is one of multiple top-level generators, its
// template will be merged with the top-level generator before the parameters are applied.
type UnionGenerator struct {
	Generators []ApplicationSetNestedGenerator `json:"generators"`
	MergeKeys  []string                        `json:"mergeKeys"`
	Template   ApplicationSetTemplate          `json:"template,omitempty"`
}

// NestedUnionGenerator is a UnionGenerator nested under another combination-type generator (MatrixGenerator or
// UnionGenerator). NestedUnionGenerator does not have an override template, because template overriding has no meaning
// within the constituent generators of combination-type generators.
type NestedUnionGenerator struct {
	Generators ApplicationSetTerminalGenerators `json:"generators"`
	MergeKeys  []string                         `json:"mergeKeys"`
}

// ToUnionGenerator converts a NestedUnionGenerator to a UnionGenerator. This conversion is for convenience, allowing
// a NestedUnionGenerator to be used where a UnionGenerator is expected (of course, the converted generator will have
// no override template).
func (g NestedUnionGenerator) ToUnionGenerator() *UnionGenerator {
	return &UnionGenerator{
		Generators: g.Generators.toApplicationSetNestedGenerators(),
		MergeKeys:  g.MergeKeys,
	}
}

// ClusterGenerator defines a generator to match against clusters registered with ArgoCD.
type ClusterGenerator struct {
	// Selector defines a label selector to match against all clusters registered with ArgoCD.
	// Clusters today are stored as Kubernetes Secrets, thus the Secret labels will be used
	// for matching the selector.
	Selector metav1.LabelSelector   `json:"selector,omitempty"`
	Template ApplicationSetTemplate `json:"template,omitempty"`

	// Values contains key/value pairs which are passed directly as parameters to the template
	Values map[string]string `json:"values,omitempty"`
}

// DuckType defines a generator to match against clusters registered with ArgoCD.
type DuckTypeGenerator struct {
	// ConfigMapRef is a ConfigMap with the duck type definitions needed to retrieve the data
	//              this includes apiVersion(group/version), kind, matchKey and validation settings
	// Name is the resource name of the kind, group and version, defined in the ConfigMapRef
	// RequeueAfterSeconds is how long before the duckType will be rechecked for a change
	ConfigMapRef        string               `json:"configMapRef"`
	Name                string               `json:"name,omitempty"`
	RequeueAfterSeconds *int64               `json:"requeueAfterSeconds,omitempty"`
	LabelSelector       metav1.LabelSelector `json:"labelSelector,omitempty"`

	Template ApplicationSetTemplate `json:"template,omitempty"`
	// Values contains key/value pairs which are passed directly as parameters to the template
	Values map[string]string `json:"values,omitempty"`
}

type GitGenerator struct {
	RepoURL             string                      `json:"repoURL"`
	Directories         []GitDirectoryGeneratorItem `json:"directories,omitempty"`
	Files               []GitFileGeneratorItem      `json:"files,omitempty"`
	Revision            string                      `json:"revision"`
	RequeueAfterSeconds *int64                      `json:"requeueAfterSeconds,omitempty"`
	Template            ApplicationSetTemplate      `json:"template,omitempty"`
}

type GitDirectoryGeneratorItem struct {
	Path    string `json:"path"`
	Exclude bool   `json:"exclude,omitempty"`
}

type GitFileGeneratorItem struct {
	Path string `json:"path"`
}

// SCMProviderGenerator defines a generator that scrapes a SCMaaS API to find candidate repos.
type SCMProviderGenerator struct {
	// Which provider to use and config for it.
	Github *SCMProviderGeneratorGithub `json:"github,omitempty"`
	Gitlab *SCMProviderGeneratorGitlab `json:"gitlab,omitempty"`
	// Filters for which repos should be considered.
	Filters []SCMProviderGeneratorFilter `json:"filters,omitempty"`
	// Which protocol to use for the SCM URL. Default is provider-specific but ssh if possible. Not all providers
	// necessarily support all protocols.
	CloneProtocol string `json:"cloneProtocol,omitempty"`
	// Standard parameters.
	RequeueAfterSeconds *int64                 `json:"requeueAfterSeconds,omitempty"`
	Template            ApplicationSetTemplate `json:"template,omitempty"`
}

// SCMProviderGeneratorGithub defines a connection info specific to GitHub.
type SCMProviderGeneratorGithub struct {
	// GitHub org to scan. Required.
	Organization string `json:"organization"`
	// The GitHub API URL to talk to. If blank, use https://api.github.com/.
	API string `json:"api,omitempty"`
	// Authentication token reference.
	TokenRef *SecretRef `json:"tokenRef,omitempty"`
	// Scan all branches instead of just the default branch.
	AllBranches bool `json:"allBranches,omitempty"`
}

// SCMProviderGeneratorGitlab defines a connection info specific to Gitlab.
type SCMProviderGeneratorGitlab struct {
	// Gitlab group to scan. Required.  You can use either the project id (recommended) or the full namespaced path.
	Group string `json:"group"`
	// Recurse through subgroups (true) or scan only the base group (false).  Defaults to "false"
	IncludeSubgroups bool `json:"includeSubgroups,omitempty"`
	// The Gitlab API URL to talk to.
	API string `json:"api,omitempty"`
	// Authentication token reference.
	TokenRef *SecretRef `json:"tokenRef,omitempty"`
	// Scan all branches instead of just the default branch.
	AllBranches bool `json:"allBranches,omitempty"`
}

// SCMProviderGeneratorFilter is a single repository filter.
// If multiple filter types are set on a single struct, they will be AND'd together. All filters must
// pass for a repo to be included.
type SCMProviderGeneratorFilter struct {
	// A regex for repo names.
	RepositoryMatch *string `json:"repositoryMatch,omitempty"`
	// An array of paths, all of which must exist.
	PathsExist []string `json:"pathsExist,omitempty"`
	// A regex which must match at least one label.
	LabelMatch *string `json:"labelMatch,omitempty"`
	// A regex which must match the branch name.
	BranchMatch *string `json:"branchMatch,omitempty"`
}

// PullRequestGenerator defines a generator that scrapes a PullRequest API to find candidate pull requests.
type PullRequestGenerator struct {
	// Which provider to use and config for it.
	Github *PullRequestGeneratorGithub `json:"github,omitempty"`
	// Standard parameters.
	RequeueAfterSeconds *int64                 `json:"requeueAfterSeconds,omitempty"`
	Template            ApplicationSetTemplate `json:"template,omitempty"`
}

// PullRequestGenerator defines a connection info specific to GitHub.
type PullRequestGeneratorGithub struct {
	// GitHub org or user to scan. Required.
	Owner string `json:"owner"`
	// GitHub repo name to scan. Required.
	Repo string `json:"repo"`
	// The GitHub API URL to talk to. If blank, use https://api.github.com/.
	API string `json:"api,omitempty"`
	// Authentication token reference.
	TokenRef *SecretRef `json:"tokenRef,omitempty"`
	// Labels is used to filter the PRs that you want to target
	Labels []string `json:"labels,omitempty"`
}

// ApplicationSetStatus defines the observed state of ApplicationSet
type ApplicationSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// ApplicationSetList contains a list of ApplicationSet
// +kubebuilder:object:root=true
type ApplicationSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApplicationSet{}, &ApplicationSetList{})
}

// RefreshRequired checks if the ApplicationSet needs to be refreshed
func (a *ApplicationSet) RefreshRequired() bool {
	_, found := a.Annotations[common.AnnotationGitGeneratorRefresh]
	return found
}
