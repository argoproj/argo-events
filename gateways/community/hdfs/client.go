package hdfs

import (
	"fmt"

	"github.com/colinmarc/hdfs"
	krb "gopkg.in/jcmturner/gokrb5.v5/client"
	"gopkg.in/jcmturner/gokrb5.v5/config"
	"gopkg.in/jcmturner/gokrb5.v5/credentials"
	"gopkg.in/jcmturner/gokrb5.v5/keytab"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HDFSConfig is config for HDFS
type HDFSConfig struct {
	Addresses  []string // comma-separated name nodes
	HDFSUser   string
	KrbOptions *KrbOptions
}

// KrbOptions is options for Kerberos
type KrbOptions struct {
	CCacheOptions        *CCacheOptions
	KeytabOptions        *KeytabOptions
	Config               string
	ServicePrincipalName string
}

// CCacheOptions is options for ccache
type CCacheOptions struct {
	CCache credentials.CCache
}

// KeytabOptions is options for keytab
type KeytabOptions struct {
	Keytab   keytab.Keytab
	Username string
	Realm    string
}

func getConfigMapKey(clientset kubernetes.Interface, namespace string, selector *corev1.ConfigMapKeySelector) (string, error) {
	configmap, err := clientset.CoreV1().ConfigMaps(namespace).Get(selector.Name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	for k, v := range configmap.Data {
		if k == selector.Key {
			return v, nil
		}
	}
	return "", fmt.Errorf("configmap '%s' does not have the key '%s'", selector.Name, selector.Key)
}

func getSecretKey(clientset kubernetes.Interface, namespace string, selector *corev1.SecretKeySelector) ([]byte, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(selector.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	for k, v := range secret.Data {
		if k == selector.Key {
			return v, nil
		}
	}
	return nil, fmt.Errorf("configmap '%s' does not have the key '%s'", selector.Name, selector.Key)
}

// createHDFSConfig constructs HDFSConfig
func createHDFSConfig(clientset kubernetes.Interface, namespace string, config *GatewayClientConfig) (*HDFSConfig, error) {
	var krbConfig string
	var krbOptions *KrbOptions
	var err error

	if config.KrbConfigConfigMap != nil && config.KrbConfigConfigMap.Name != "" {
		krbConfig, err = getConfigMapKey(clientset, namespace, config.KrbConfigConfigMap)
		if err != nil {
			return nil, err
		}
	}
	if config.KrbCCacheSecret != nil && config.KrbCCacheSecret.Name != "" {
		bytes, err := getSecretKey(clientset, namespace, config.KrbCCacheSecret)
		if err != nil {
			return nil, err
		}
		ccache, err := credentials.ParseCCache(bytes)
		if err != nil {
			return nil, err
		}
		krbOptions = &KrbOptions{
			CCacheOptions: &CCacheOptions{
				CCache: ccache,
			},
			Config:               krbConfig,
			ServicePrincipalName: config.KrbServicePrincipalName,
		}
	}
	if config.KrbKeytabSecret != nil && config.KrbKeytabSecret.Name != "" {
		bytes, err := getSecretKey(clientset, namespace, config.KrbKeytabSecret)
		if err != nil {
			return nil, err
		}
		ktb, err := keytab.Parse(bytes)
		if err != nil {
			return nil, err
		}
		krbOptions = &KrbOptions{
			KeytabOptions: &KeytabOptions{
				Keytab:   ktb,
				Username: config.KrbUsername,
				Realm:    config.KrbRealm,
			},
			Config:               krbConfig,
			ServicePrincipalName: config.KrbServicePrincipalName,
		}
	}

	hdfsConfig := HDFSConfig{
		Addresses:  config.Addresses,
		HDFSUser:   config.HDFSUser,
		KrbOptions: krbOptions,
	}
	return &hdfsConfig, nil
}

func createHDFSClient(addresses []string, user string, krbOptions *KrbOptions) (*hdfs.Client, error) {
	options := hdfs.ClientOptions{
		Addresses: addresses,
	}

	if krbOptions != nil {
		krbClient, err := createKrbClient(krbOptions)
		if err != nil {
			return nil, err
		}
		options.KerberosClient = krbClient
		options.KerberosServicePrincipleName = krbOptions.ServicePrincipalName
	} else {
		options.User = user
	}

	return hdfs.NewClient(options)
}

func createKrbClient(krbOptions *KrbOptions) (*krb.Client, error) {
	krbConfig, err := config.NewConfigFromString(krbOptions.Config)
	if err != nil {
		return nil, err
	}

	if krbOptions.CCacheOptions != nil {
		client, err := krb.NewClientFromCCache(krbOptions.CCacheOptions.CCache)
		if err != nil {
			return nil, err
		}
		return client.WithConfig(krbConfig), nil
	} else if krbOptions.KeytabOptions != nil {
		client := krb.NewClientWithKeytab(krbOptions.KeytabOptions.Username, krbOptions.KeytabOptions.Realm, krbOptions.KeytabOptions.Keytab)
		client = *client.WithConfig(krbConfig)
		err = client.Login()
		if err != nil {
			return nil, err
		}
		return &client, nil
	}

	return nil, fmt.Errorf("Failed to get a Kerberos client")
}
