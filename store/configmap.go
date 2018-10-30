package store

import (
	"fmt"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ConfigmapReader implements the ArtifactReader interface for k8 configmap
type ConfigmapReader struct {
	kubeClientset     kubernetes.Interface
	configmapArtifact *v1alpha1.ConfigmapArtifact
}

type configmapWf struct {
	wf wfv1.Workflow
}

// NewConfigmapReader returns a new configmap reader
func NewConfigmapReader(kubeClientset kubernetes.Interface, configmapArtifact *v1alpha1.ConfigmapArtifact) (*ConfigmapReader, error) {
	return &ConfigmapReader{
		kubeClientset:     kubeClientset,
		configmapArtifact: configmapArtifact,
	}, nil
}

func (c *ConfigmapReader) Read() (body []byte, err error) {
	cm, err := c.kubeClientset.CoreV1().ConfigMaps(c.configmapArtifact.Namespace).Get(c.configmapArtifact.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if resource, ok := cm.Data[c.configmapArtifact.Key]; ok {
		return []byte(resource), nil
	}
	return nil, fmt.Errorf("unable to find key %s in configmap %s", c.configmapArtifact.Key, c.configmapArtifact.Name)
}
