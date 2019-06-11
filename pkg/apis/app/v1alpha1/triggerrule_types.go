package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TriggerRuleSpec defines the desired state of TriggerRule
// +k8s:openapi-gen=true
type TriggerRuleSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Sources []Source `json:"sources,omitempty"`
	Actions []Action `json:"actions,omitempty"`
}

// Source describes the resource that can be watched for updates.
type Source struct {
	// ObjectRef specifies an kubernetes object. Trigger will watch updates and trigger related actions.
	// Object can must be ConfigMap or Secret.
	ObjectRef corev1.ObjectReference `json:"objectRef,omitempty"`
}

// Action describes what to do when update occurs.
type Action struct {
	// UpdatePodTemplate will trigger workload rolling update by updating a special annotation of pod template.
	UpdatePodTemplate *ActionUpdatePodTemplate `json:"updatePodTemplate,omitempty"`
}

type ActionUpdatePodTemplate struct {
	ObjectRef corev1.ObjectReference `json:"objectRef,omitempty"`
}

// TriggerRuleStatus defines the observed state of TriggerRule
// +k8s:openapi-gen=true
type TriggerRuleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerRule is the Schema for the triggerrules API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type TriggerRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TriggerRuleSpec   `json:"spec,omitempty"`
	Status TriggerRuleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TriggerRuleList contains a list of TriggerRule
type TriggerRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TriggerRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TriggerRule{}, &TriggerRuleList{})
}
