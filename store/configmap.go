package store

import (
	"fmt"
	"os"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ConfigMapReader implements the ArtifactReader interface for k8 configmap
type ConfigMapReader struct {
	kubeClientset     kubernetes.Interface
	configmapArtifact *v1alpha1.ConfigmapArtifact
}

// NewConfigMapReader returns a new configmap reader
func NewConfigMapReader(kubeClientset kubernetes.Interface, configmapArtifact *v1alpha1.ConfigmapArtifact) (*ConfigMapReader, error) {
	return &ConfigMapReader{
		kubeClientset:     kubeClientset,
		configmapArtifact: configmapArtifact,
	}, nil
}

func (c *ConfigMapReader) Read() (body []byte, err error) {
	namespace := os.Getenv(common.EnvVarNamespace)
	if c.configmapArtifact.Namespace != "" {
		namespace = c.configmapArtifact.Namespace
	}
	cm, err := c.kubeClientset.CoreV1().ConfigMaps(namespace).Get(c.configmapArtifact.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if resource, ok := cm.Data[c.configmapArtifact.Key]; ok {
		return []byte(resource), nil
	}
	return nil, fmt.Errorf("unable to find key %s in configmap %s", c.configmapArtifact.Key, c.configmapArtifact.Name)
}
