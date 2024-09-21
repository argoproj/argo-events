package v1alpha1

import corev1 "k8s.io/api/core/v1"

// TLSConfig refers to TLS configuration for a client.
type TLSConfig struct {
	// CACertSecret refers to the secret that contains the CA cert
	CACertSecret *corev1.SecretKeySelector `json:"caCertSecret,omitempty" protobuf:"bytes,1,opt,name=caCertSecret"`
	// ClientCertSecret refers to the secret that contains the client cert
	ClientCertSecret *corev1.SecretKeySelector `json:"clientCertSecret,omitempty" protobuf:"bytes,2,opt,name=clientCertSecret"`
	// ClientKeySecret refers to the secret that contains the client key
	ClientKeySecret *corev1.SecretKeySelector `json:"clientKeySecret,omitempty" protobuf:"bytes,3,opt,name=clientKeySecret"`
	// If true, skips creation of TLSConfig with certs and creates an empty TLSConfig. (Defaults to false)
	// +optional
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty" protobuf:"varint,4,opt,name=insecureSkipVerify"`
}
