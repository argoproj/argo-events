package hdfs

import (
	"github.com/ghodss/yaml"
	"github.com/rs/zerolog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// EventSourceExecutor implements Eventing
type EventSourceExecutor struct {
	Log zerolog.Logger
	// Clientset is kubernetes client
	Clientset kubernetes.Interface
	// Namespace where gateway is deployed
	Namespace string
}

// GatewayConfig contains information to setup a HDFS integration
type GatewayConfig struct {
	// Directory to watch for events
	Directory string `json:"directory"`
	// Path is relative path of object to watch with respect to the directory
	Path string `json:"path"`
	// Type of file operations to watch
	Type string `json:"type"`
	// CheckInterval is a string that describes an interval duration to check the directory state, e.g. 1s, 30m, 2h... (defaults to 1m)
	CheckInterval string `json:"checkInterval,omitempty"`

	GatewayClientConfig `json:",inline"`
}

// GatewayClientConfig contains HDFS client configurations
type GatewayClientConfig struct {
	// Addresses is accessible addresses of HDFS name nodes
	Addresses []string `json:"addresses"`

	// HDFSUser is the user to access HDFS file system.
	// It is ignored if either ccache or keytab is used.
	HDFSUser string `json:"hdfsUser,omitempty"`

	// KrbCCacheSecret is the secret selector for Kerberos ccache
	// Either ccache or keytab can be set to use Kerberos.
	KrbCCacheSecret *corev1.SecretKeySelector `json:"krbCCacheSecret,omitempty"`

	// KrbKeytabSecret is the secret selector for Kerberos keytab
	// Either ccache or keytab can be set to use Kerberos.
	KrbKeytabSecret *corev1.SecretKeySelector `json:"krbKeytabSecret,omitempty"`

	// KrbUsername is the Kerberos username used with Kerberos keytab
	// It must be set if keytab is used.
	KrbUsername string `json:"krbUsername,omitempty"`

	// KrbRealm is the Kerberos realm used with Kerberos keytab
	// It must be set if keytab is used.
	KrbRealm string `json:"krbRealm,omitempty"`

	// KrbConfig is the configmap selector for Kerberos config as string
	// It must be set if either ccache or keytab is used.
	KrbConfigConfigMap *corev1.ConfigMapKeySelector `json:"krbConfigConfigMap,omitempty"`

	// KrbServicePrincipalName is the principal name of Kerberos service
	// It must be set if either ccache or keytab is used.
	KrbServicePrincipalName string `json:"krbServicePrincipalName,omitempty"`
}

func parseEventSource(eventSource string) (interface{}, error) {
	var f *GatewayConfig
	err := yaml.Unmarshal([]byte(eventSource), &f)
	if err != nil {
		return nil, err
	}
	return f, err
}
