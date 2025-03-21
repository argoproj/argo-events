package v1alpha1

import corev1 "k8s.io/api/core/v1"

// ContainerTemplate defines customized spec for a container
type ContainerTemplate struct {
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,1,opt,name=resources"`
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty" protobuf:"bytes,2,opt,name=imagePullPolicy,casttype=PullPolicy"`
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty" protobuf:"bytes,3,opt,name=securityContext"`
}
