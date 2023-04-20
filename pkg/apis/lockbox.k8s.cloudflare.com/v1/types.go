package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="SecretType",type=string,JSONPath=`.spec.template.type`
// +kubebuilder:printcolumn:name="Peer",type=string,JSONPath=`.spec.peer`

// Lockbox is a struct wrapping the LockboxSpec in standard API server
// metadata fields.
type Lockbox struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Desired state of the Lockbox resource.
	Spec LockboxSpec `json:"spec"`

	// Status of the Lockbox. This is set and managed automatically.
	// +optional
	Status LockboxStatus `json:"status,omitempty"`
}

// LockboxSpec is a struct wrapping the encrypted secrets along with the
// public keys of the sender and server.
type LockboxSpec struct {
	// Sender stores the public key used to lock this Lockbox.
	Sender []byte `json:"sender"`

	// Peer stores the public key that can unlock this Lockbox.
	Peer []byte `json:"peer"`

	// Namespace stores an encrypted copy of which namespace this Lockbox is locked
	// for, ensuring it cannot be deployed to another namespace under an attacker's
	// control.
	Namespace []byte `json:"namespace"`

	// Data contains the secret data, encrypted to the Peer's public key. Each key in the
	// data map must consist of alphanumeric characters, '-', '_', or '.'.
	Data map[string][]byte `json:"data"`

	// Template defines the structure of the Secret that will be
	// created from this Lockbox.
	// +optional
	Template LockboxSecretTemplate `json:"template,omitempty"`
}

// LockboxSecretTemplate defines structure of API metadata fields
// of Secrets controlled by a Lockbox.
type LockboxSecretTemplate struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Type is used to facilitate programmatic handling of secret data.
	Type corev1.SecretType `json:"type,omitempty"`
}

// LockboxStatus contains status information about a Lockbox.
type LockboxStatus struct {
	// List of status conditions to indicate the status of a Lockbox.
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`
}

// Condition contains condition information for a Lockbox.
type Condition struct {
	// Type of condition in CamelCase.
	// +required
	Type ConditionType `json:"type"`

	// Status of the condition, one of True, False, Unknown
	// +required
	Status corev1.ConditionStatus `json:"status"`

	// Severity provides explicit classification of Reason code, so that users or machines
	// can immediately understand the current situation and act accordingly.
	// The Severity field MUST be set only when Status=False.
	// +optional
	Severity ConditionSeverity `json:"severity"`

	// LastTransitionTime marks when the condition last transitioned from one status to another.
	// This should be when the underlying condition changed. If that is not known, then using the time
	// when the API field changed is acceptable.
	// +required
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// The reason for the condition's last transition in CamelCase.
	// +optional
	Reason string `json:"reason,omitempty"`

	// A message is the human readable message indicating details about the transition.
	// The field may be empty.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:validation:Enum=Ready
type ConditionType string

const (
	ReadyCondition ConditionType = "Ready"
)

// +kubebuilder:validation:Enum=Error;Warning;Info
type ConditionSeverity string

const (
	ConditionSeverityError   ConditionSeverity = "Error"
	ConditionSeverityWarning ConditionSeverity = "Warning"
	ConditionSeverityInfo    ConditionSeverity = "Info"
	ConditionSeverityNone    ConditionSeverity = ""
)

// +kubebuilder:object:root=k8s.io/apimachinery/pkg/runtime.Object

// LockboxList is a Lockbox-specific version of metav1.List.
type LockboxList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []Lockbox
}
