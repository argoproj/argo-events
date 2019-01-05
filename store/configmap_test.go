package store

import (
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestConfigmapReader_Read(t *testing.T) {
	kubeClientset := fake.NewSimpleClientset()
	key := "wf"
	cmArtifact := &v1alpha1.ConfigmapArtifact{
		Name:      "wf-configmap",
		Namespace: "argo-events",
		Key:       key,
	}

	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmArtifact.Name,
			Namespace: cmArtifact.Namespace,
		},
		Data: map[string]string{
			key: `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
       args:
       - "hello world"	
       command:
       - cowsay
       image: "docker/whalesay:latest"`,
		},
	}

	cm, err := kubeClientset.CoreV1().ConfigMaps(cmArtifact.Namespace).Create(configmap)
	assert.Nil(t, err)
	assert.Equal(t, cm.Name, configmap.Name)
	cmReader, err := NewConfigMapReader(kubeClientset, cmArtifact)
	assert.Nil(t, err)
	resourceBody, err := cmReader.Read()
	assert.Nil(t, err)

	var wf *wfv1.Workflow
	err = yaml.Unmarshal(resourceBody, &wf)
	assert.Nil(t, err)
	assert.Equal(t, wf.Name, "hello-world")
}
