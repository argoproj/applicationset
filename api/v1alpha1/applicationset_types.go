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
	"fmt"
	"sort"

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
// +kubebuilder:subresource:status
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

// ApplicationSetGenerator include list item info
type ApplicationSetGenerator struct {
	List                    *ListGenerator        `json:"list,omitempty"`
	Clusters                *ClusterGenerator     `json:"clusters,omitempty"`
	Git                     *GitGenerator         `json:"git,omitempty"`
	Matrix                  *MatrixGenerator      `json:"matrix,omitempty"`
	SCMProvider             *SCMProviderGenerator `json:"scmProvider,omitempty"`
	ClusterDecisionResource *DuckTypeGenerator    `json:"clusterDecisionResource,omitempty"`
	PullRequest             *PullRequestGenerator `json:"pullRequest,omitempty"`
}

// ApplicationSetBaseGenerator include list item info
// CRD dosn't support recursive types so we need a different type for the matrix generator
// https://github.com/kubernetes-sigs/controller-tools/issues/477
type ApplicationSetBaseGenerator struct {
	List                    *ListGenerator        `json:"list,omitempty"`
	Clusters                *ClusterGenerator     `json:"clusters,omitempty"`
	Git                     *GitGenerator         `json:"git,omitempty"`
	SCMProvider             *SCMProviderGenerator `json:"scmProvider,omitempty"`
	ClusterDecisionResource *DuckTypeGenerator    `json:"clusterDecisionResource,omitempty"`
	PullRequest             *PullRequestGenerator `json:"pullRequest,omitempty"`
}

// ListGenerator include items info
type ListGenerator struct {
	Elements []apiextensionsv1.JSON `json:"elements"`
	Template ApplicationSetTemplate `json:"template,omitempty"`
}

// MatrixGenerator include Other generators
type MatrixGenerator struct {
	Generators []ApplicationSetBaseGenerator `json:"generators"`
	Template   ApplicationSetTemplate        `json:"template,omitempty"`
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
	// ConfigMapRef is a ConfigMap with the duck type definitions needed to retreive the data
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
	Conditions []ApplicationSetCondition `json:"conditions,omitempty"`
}

// ApplicationSetCondition contains details about an applicationset condition, which is usally an error or warning
type ApplicationSetCondition struct {
	// Type is an applicationset condition type
	Type ApplicationSetConditionType `json:"type" protobuf:"bytes,1,opt,name=type"`
	// Message contains human-readable message indicating details about condition
	Message string `json:"message" protobuf:"bytes,2,opt,name=message"`
	// LastTransitionTime is the time the condition was last observed
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`
	// True/False/Unknown
	Status ApplicationSetConditionStatus `json:"status" protobuf:"bytes,4,opt,name=status"`
	//Single word camelcase representing the reason for the status eg ErrorOccurred
	Reason string `json:"reason" protobuf:"bytes,5,opt,name=reason"`
}

// SyncStatusCode is a type which represents possible comparison results
type ApplicationSetConditionStatus string

// Application Condition Status
const (
	// ApplicationSetConditionStatusTrue indicates that a application has been successfully established
	ApplicationSetConditionStatusTrue ApplicationSetConditionStatus = "True"
	// ApplicationSetConditionStatusFalse indicates that a application attempt has failed
	ApplicationSetConditionStatusFalse ApplicationSetConditionStatus = "False"
	// ApplicationSetConditionStatusUnknown indicates that the application condition status could not be reliably determined
	ApplicationSetConditionStatusUnknown ApplicationSetConditionStatus = "Unknown"
)

// ApplicationSetConditionType represents type of application condition. Type name has following convention:
// prefix "Error" means error condition
// prefix "Warning" means warning condition
// prefix "Info" means informational condition
type ApplicationSetConditionType string

//ErrorOccurred / ParametersGenerated / TemplateRendered / ResourcesUpToDate
const (
	ApplicationSetConditionErrorOccured        ApplicationSetConditionType = "ErrorOccured"
	ApplicationSetConditionParametersGenerated ApplicationSetConditionType = "ParametersGenerated"
	ApplicationSetConditionResourcesUpToDate   ApplicationSetConditionType = "ResourcesUpToDate"
)

const (
	// ApplicationSetReferencedProjectNotFound                  = "ReferencedProjectNotFound"
	// ApplicationSetReasonInvalidApplicationSpec               = "InvalidApplicationSpec"
	ApplicationSetReasonErrorOccured           = "ErrorOccured"
	ApplicationSetReasonApplicationSetUpToDate = "ApplicationSetUpToDate"
	ApplicationSetReasonUpdateApplicationError = "UpdateApplicationError"
	// ApplicationSetReasonApplicationsWithDuplicateNames       = "ApplicationsWithDuplicateNames"
	ApplicationSetReasonApplicationGenerationFromParamsError = "ApplicationGenerationFromParamsError"
	ApplicationSetReasonRenderTemplateParamsError            = "RenderTemplateParamsError"
	ApplicationSetReasonCreateApplicationError               = "CreateApplicationError"
	ApplicationSetReasonDeleteApplicationError               = "DeleteApplicationError"
	ApplicationSetReasonRefreshApplicationError              = "RefreshApplicationError"
	ApplicationSetReasonApplicationValidationError           = "ApplicationValidationError"
)

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

// SetConditions updates the applicationset status conditions for a subset of evaluated types.
// If the applicationset has a pre-existing condition of a type that is not in the evaluated list,
// it will be preserved. If the applicationset has a pre-existing condition of a type, status, reason that
// is in the evaluated list, but not in the incoming conditions list, it will be removed.
func (status *ApplicationSetStatus) SetConditions(conditions []ApplicationSetCondition, evaluatedTypes map[ApplicationSetConditionType]bool) {
	applicationSetConditions := make([]ApplicationSetCondition, 0)
	now := metav1.Now()
	for i := 0; i < len(status.Conditions); i++ {
		condition := status.Conditions[i]
		if _, ok := evaluatedTypes[condition.Type]; !ok {
			if condition.LastTransitionTime == nil {
				condition.LastTransitionTime = &now
			}
			applicationSetConditions = append(applicationSetConditions, condition)
		}
	}
	for i := range conditions {
		condition := conditions[i]
		if condition.LastTransitionTime == nil {
			condition.LastTransitionTime = &now
		}
		eci := findConditionIndex(status.Conditions, condition.Type, condition.Status, condition.Reason)
		if eci >= 0 && status.Conditions[eci].Message == condition.Message {
			// If we already have a condition of this type, status and reason, only update the timestamp if something
			// has changed.
			applicationSetConditions = append(applicationSetConditions, status.Conditions[eci])
		} else {
			// Otherwise we use the new incoming condition with an updated timestamp:
			applicationSetConditions = append(applicationSetConditions, condition)
		}
	}
	sort.Slice(applicationSetConditions, func(i, j int) bool {
		left := applicationSetConditions[i]
		right := applicationSetConditions[j]
		return fmt.Sprintf("%s/%s/%v", left.Type, left.Message, left.LastTransitionTime) < fmt.Sprintf("%s/%s/%v", right.Type, right.Message, right.LastTransitionTime)
	})
	status.Conditions = applicationSetConditions
}

func findConditionIndex(conditions []ApplicationSetCondition, t ApplicationSetConditionType, status ApplicationSetConditionStatus, reason string) int {
	for i := range conditions {
		if conditions[i].Type == t && conditions[i].Status == status && conditions[i].Reason == reason {
			return i
		}
	}
	return -1
}
