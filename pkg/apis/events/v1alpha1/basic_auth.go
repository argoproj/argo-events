package v1alpha1

import corev1 "k8s.io/api/core/v1"

// BasicAuth contains the reference to K8s secrets that holds the username and password
type BasicAuth struct {
	// Username refers to the Kubernetes secret that holds the username required for basic auth.
	Username *corev1.SecretKeySelector `json:"username,omitempty" protobuf:"bytes,1,opt,name=username"`
	// Password refers to the Kubernetes secret that holds the password required for basic auth.
	Password *corev1.SecretKeySelector `json:"password,omitempty" protobuf:"bytes,2,opt,name=password"`
}
