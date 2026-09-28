// +kubebuilder:object:generate=true
// +groupName=lockbox.k8s.cloudflare.com

// Package v1 is the v1 version of the Lockbox API
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//go:generate go tool controller-gen object crd paths=./. output:crd:artifacts:config=../../../../deployment/crds

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "lockbox.k8s.cloudflare.com", Version: "v1"}
	GroupVersion       = SchemeGroupVersion

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion, &Lockbox{}, &LockboxList{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)

	return nil
}
