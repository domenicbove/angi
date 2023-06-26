package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MyAppResourceSpec defines the desired state of MyAppResource
type MyAppResourceSpec struct {
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// ReplicaCount sets the pod replicas for the PodInfo Deployment.
	ReplicaCount *int32 `json:"replicaCount,omitempty"`

	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// +optional
	Image *Image `json:"image,omitempty"`

	UI UI `json:"ui"`

	// +optional
	Redis *Redis `json:"redis,omitempty"`
}

// Image describes the PodInfo Container image.
type Image struct {
	// +optional
	// +kubebuilder:default=ghcr.io/stefanprodan/podinfo
	// Repository sets the PodInfo Container image repository.
	Repository string `json:"repository,omitempty"`

	// +optional
	// +kubebuilder:default=latest
	// Tag sets the PodInfo Container image tag.
	Tag string `json:"tag,omitempty"`
}

// UI describes the PodInfo Container UI settings.
type UI struct {
	// +kubebuilder:validation:Pattern=`^#[A-Fa-f0-9]{6}`
	// Repository sets the PodInfo UI color.
	Color string `json:"color"`

	// Message sets the PodInfo UI message.
	Message string `json:"message"`
}

// Redis describes the Redis Deployment.
type Redis struct {
	// Enabled specifies to deploy a backing redis deployment.
	Enabled bool `json:"enabled"`
}

// MyAppResourceStatus defines the observed state of MyAppResource
type MyAppResourceStatus struct {
	// +optional
	// PodInfoReadyReplicas is the number of pods targeted by the PodInfo Deployment with a Ready Condition.
	PodInfoReadyReplicas int32 `json:"podInfoReadyReplicas,omitempty"`
	// +optional
	// RedisReadyReplicas is the number of pods targeted by the Redis Deployment with a Ready Condition.
	RedisReadyReplicas int32 `json:"redisReadyReplicas,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MyAppResource is the Schema for the myappresources API
type MyAppResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MyAppResourceSpec   `json:"spec,omitempty"`
	Status MyAppResourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MyAppResourceList contains a list of MyAppResource
type MyAppResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyAppResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MyAppResource{}, &MyAppResourceList{})
}
