package hdfs

import (
	"fmt"

	"github.com/colinmarc/hdfs/v2"
	krb "github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/keytab"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
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
	CCache *credentials.CCache
}

// KeytabOptions is options for keytab
type KeytabOptions struct {
	Keytab   *keytab.Keytab
	Username string
	Realm    string
}

func getConfigMapKey(selector *corev1.ConfigMapKeySelector) (string, error) {
	result, err := common.GetConfigMapFromVolume(selector)
	if err != nil {
		return "", fmt.Errorf("configmap value not injected, %w", err)
	}
	return result, nil
}

func getSecretKey(selector *corev1.SecretKeySelector) ([]byte, error) {
	result, err := common.GetSecretFromVolume(selector)
	if err != nil {
		return nil, fmt.Errorf("secret value not injected, %w", err)
	}
	return []byte(result), nil
}

// createHDFSConfig constructs HDFSConfig
func createHDFSConfig(hdfsEventSource *v1alpha1.HDFSEventSource) (*HDFSConfig, error) {
	var krbConfig string
	var krbOptions *KrbOptions
	var err error

	if hdfsEventSource.KrbConfigConfigMap != nil && hdfsEventSource.KrbConfigConfigMap.Name != "" {
		krbConfig, err = getConfigMapKey(hdfsEventSource.KrbConfigConfigMap)
		if err != nil {
			return nil, err
		}
	}
	if hdfsEventSource.KrbCCacheSecret != nil && hdfsEventSource.KrbCCacheSecret.Name != "" {
		bytes, err := getSecretKey(hdfsEventSource.KrbCCacheSecret)
		if err != nil {
			return nil, err
		}
		ccache := new(credentials.CCache)
		err = ccache.Unmarshal(bytes)
		if err != nil {
			return nil, err
		}
		krbOptions = &KrbOptions{
			CCacheOptions: &CCacheOptions{
				CCache: ccache,
			},
			Config:               krbConfig,
			ServicePrincipalName: hdfsEventSource.KrbServicePrincipalName,
		}
	}
	if hdfsEventSource.KrbKeytabSecret != nil && hdfsEventSource.KrbKeytabSecret.Name != "" {
		bytes, err := getSecretKey(hdfsEventSource.KrbKeytabSecret)
		if err != nil {
			return nil, err
		}
		ktb := new(keytab.Keytab)
		err = ktb.Unmarshal(bytes)
		if err != nil {
			return nil, err
		}
		krbOptions = &KrbOptions{
			KeytabOptions: &KeytabOptions{
				Keytab:   ktb,
				Username: hdfsEventSource.KrbUsername,
				Realm:    hdfsEventSource.KrbRealm,
			},
			Config:               krbConfig,
			ServicePrincipalName: hdfsEventSource.KrbServicePrincipalName,
		}
	}

	hdfsConfig := HDFSConfig{
		Addresses:  hdfsEventSource.Addresses,
		HDFSUser:   hdfsEventSource.HDFSUser,
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
	krbConfig, err := config.NewFromString(krbOptions.Config)
	if err != nil {
		return nil, err
	}

	if krbOptions.CCacheOptions != nil {
		return krb.NewFromCCache(krbOptions.CCacheOptions.CCache, krbConfig)
	} else if krbOptions.KeytabOptions != nil {
		client := krb.NewWithKeytab(krbOptions.KeytabOptions.Username, krbOptions.KeytabOptions.Realm, krbOptions.KeytabOptions.Keytab, krbConfig)
		err = client.Login()
		if err != nil {
			return nil, err
		}
		return client, nil
	}

	return nil, fmt.Errorf("Failed to get a Kerberos client")
}
