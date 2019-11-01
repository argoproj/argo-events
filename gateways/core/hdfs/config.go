package hdfs

import (
	"errors"
	"github.com/sirupsen/logrus"

	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const ArgoEventsEventSourceVersion = "v0.10"

// EventSourceExecutor implements Eventing
type EventSourceExecutor struct {
	Log *logrus.Logger
	// k8sClient is kubernetes client
	Clientset kubernetes.Interface
	// Namespace where gateway is deployed
	Namespace string
}

// GatewayConfig contains information to setup a HDFS integration
type GatewayConfig struct {
	gwcommon.WatchPathConfig `json:",inline"`

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

// Validate validates GatewayClientConfig
func (c *GatewayClientConfig) Validate() error {
	if len(c.Addresses) == 0 {
		return errors.New("addresses is required")
	}

	hasKrbCCache := c.KrbCCacheSecret != nil
	hasKrbKeytab := c.KrbKeytabSecret != nil

	if c.HDFSUser == "" && !hasKrbCCache && !hasKrbKeytab {
		return errors.New("either hdfsUser, krbCCacheSecret or krbKeytabSecret is required")
	}
	if hasKrbKeytab && (c.KrbServicePrincipalName == "" || c.KrbConfigConfigMap == nil || c.KrbUsername == "" || c.KrbRealm == "") {
		return errors.New("krbServicePrincipalName, krbConfigConfigMap, krbUsername and krbRealm are required with krbKeytabSecret")
	}
	if hasKrbCCache && (c.KrbServicePrincipalName == "" || c.KrbConfigConfigMap == nil) {
		return errors.New("krbServicePrincipalName and krbConfigConfigMap are required with krbCCacheSecret")
	}

	return nil
}
