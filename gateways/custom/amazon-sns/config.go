package amazon_sns

import (
	corev1 "k8s.io/api/core/v1"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/argoproj/argo-events/gateways"
	"github.com/ghodss/yaml"
)

// AWSSNSConfig contains information to configure sns notifications
type AWSSNSConfig struct {
	// AccessKey refers to k8 secret containing aws access key
	AccessKey *corev1.SecretKeySelector

	// SecretKey refers to k8 secret containing aws secret key
	SecretKey *corev1.SecretKeySelector

	// AccessToken refers to k8 secret containing aws access token
	AccessToken *corev1.SecretKeySelector

	// Region is the aws region
	Region string
}

// AWSSNSConfigExecutor implements ConfigExecutor
type AWSSNSConfigExecutor struct {
	*gateways.GatewayConfig
	snsClient *sns.SNS
}


// parseConfig parses a configuration of gateway
func parseConfig(config string) (*AWSSNSConfig, error) {
	var a *AWSSNSConfig
	err := yaml.Unmarshal([]byte(config), &a)
	if err != nil {
		return nil, err
	}
	return a, err
}
