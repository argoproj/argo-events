package store

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ConfigMapReader implements the ArtifactReader interface for k8 configmap
type ConfigMapReader struct {
	kubeClientset     kubernetes.Interface
	configmapArtifact *corev1.ConfigMapKeySelector
	namespace         string
}

// NewConfigMapReader returns a new configmap reader
func NewConfigMapReader(kubeClientset kubernetes.Interface, namespace string, configmapArtifact *corev1.ConfigMapKeySelector) (*ConfigMapReader, error) {
	return &ConfigMapReader{
		kubeClientset:     kubeClientset,
		configmapArtifact: configmapArtifact,
		namespace:         namespace,
	}, nil
}

func (c *ConfigMapReader) Read() (body []byte, err error) {
	cm, err := c.kubeClientset.CoreV1().ConfigMaps(c.namespace).Get(c.configmapArtifact.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if resource, ok := cm.Data[c.configmapArtifact.Key]; ok {
		return []byte(resource), nil
	}
	return nil, fmt.Errorf("unable to find key %s in configmap %s", c.configmapArtifact.Key, c.configmapArtifact.Name)
}
