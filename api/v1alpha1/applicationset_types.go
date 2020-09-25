package v1alpha1

import (
	"github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplicationSet is a set of Application resources
// +kubebuilder:object:root=true
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApplicationSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ApplicationSetSpec `json:"spec"`
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
	// SkipPrune will disable the default behavior which will delete Applications that are no longer being generated for the ApplicationSet which created them, or the ApplicationSet itself is deleted. If SkipPrune is set to true, these Applications will be orphaned but continue to exist.
	SkipPrune bool `json:"skipPrune,omitempty"`
}

// ApplicationSetTemplate represents argocd ApplicationSpec
type ApplicationSetTemplate struct {
	metav1.ObjectMeta `json:"metadata"`
	Spec              v1alpha1.ApplicationSpec `json:"spec"`
}

// ApplicationSetGenerator include list item info
type ApplicationSetGenerator struct {
	List     *ListGenerator    `json:"list,omitempty"`
	Clusters *ClusterGenerator `json:"clusters,omitempty"`
	Git      *GitGenerator     `json:"git,omitempty"`
}

// ListGenerator include items info
type ListGenerator struct {
	Elements []ListGeneratorElement `json:"elements"`
}

// ListGeneratorItem include cluster and url info
type ListGeneratorElement struct {
	Cluster string `json:"cluster"`
	Url     string `json:"url"`
}

// ClusterGenerator defines a generator to match against clusters registered with ArgoCD.
type ClusterGenerator struct {
	// Selector defines a label selector to match against all clusters registered with ArgoCD.
	// Clusters today are stored as Kubernetes Secrets, thus the Secret labels will be used
	// for matching the selector.
	Selector metav1.LabelSelector `json:"selector,omitempty"`
}

type GitGenerator struct {
	RepoURL             string                      `json:"repoURL"`
	Directories         []GitDirectoryGeneratorItem `json:"directories,omitempty"`
	Revision            string                      `json:"revision"`
	RequeueAfterSeconds int64                       `json:"requeueAfterSeconds,omitempty"`
}

type GitDirectoryGeneratorItem struct {
	Path string `json:"path"`
}

// +kubebuilder:object:root=true

// ApplicationSetList contains a list of ApplicationSet
type ApplicationSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApplicationSet{}, &ApplicationSetList{})
}
