package common

import corev1 "k8s.io/api/core/v1"

// TLSConfig refers to TLS configuration for a client.
type TLSConfig struct {
	// CACertSecret refers to the secret that contains the CA cert
	CACertSecret *corev1.SecretKeySelector `json:"caCertSecret,omitempty" protobuf:"bytes,1,opt,name=caCertSecret"`
	// ClientCertSecret refers to the secret that contains the client cert
	ClientCertSecret *corev1.SecretKeySelector `json:"clientCertSecret,omitempty" protobuf:"bytes,2,opt,name=clientCertSecret"`
	// ClientKeySecret refers to the secret that contains the client key
	ClientKeySecret *corev1.SecretKeySelector `json:"clientKeySecret,omitempty" protobuf:"bytes,3,opt,name=clientKeySecret"`

	// DeprecatedCACertPath refers the file path that contains the CA cert.
	// Deprecated: use CACertSecret instead
	DeprecatedCACertPath string `json:"caCertPath" protobuf:"bytes,4,opt,name=caCertPath"`
	// DeprecatedClientCertPath refers the file path that contains client cert.
	// Deprecated: use ClientCertSecret instead
	DeprecatedClientCertPath string `json:"clientCertPath" protobuf:"bytes,5,opt,name=clientCertPath"`
	// DeprecatedClientKeyPath refers the file path that contains client key.
	// Deprecated: use ClientKeySecret instead
	DeprecatedClientKeyPath string `json:"clientKeyPath" protobuf:"bytes,6,opt,name=clientKeyPath"`
}
