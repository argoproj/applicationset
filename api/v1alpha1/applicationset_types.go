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
	Generators ApplicationSetGenerators  `json:"generators"`
	Template   ApplicationSetTemplate    `json:"template"`
	Operation  *v1alpha1.SyncOperation   `json:"operation,omitempty"`
	SyncPolicy *ApplicationSetSyncPolicy `json:"syncPolicy,omitempty"`
}

// ApplicationSetSyncPolicy will provide a syncPolicy similar to Applications
type ApplicationSetSyncPolicy struct {
	// Automated will keep an application synced to the target revision
	Automated *SyncPolicyAutomated `json:"automated,omitempty"`
}

// SyncPolicyAutomated
type SyncPolicyAutomated struct {
	// Prune will prune resources automatically as part of automated sync (default: false)
	Prune       bool `json:"prune,omitempty"`
	InitialSync bool `json:"initialSync,omitempty"`
}

// ApplicationSetTemplate represents argocd ApplicationSpec
type ApplicationSetTemplate struct {
	metav1.ObjectMeta `json:"metadata"`
	TemplateSpec      v1alpha1.ApplicationSpec `json:"spec"`
}

// ApplicationSetGenerators include list item info
type ApplicationSetGenerators struct {
	List GeneratorsList `json:"list, omitempty"`
}

// GeneratorsList include items info
type GeneratorsList struct {
	Items GeneratorsItems `json:"items"`
}

// GeneratorsItems include cluster and url info
type GeneratorsItems struct {
	Cluster string `json:"cluster"`
	Url     string `json:"url"`
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
