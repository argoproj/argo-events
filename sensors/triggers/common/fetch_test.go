package common

import (
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

var sensorObj = &v1alpha1.Sensor{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "fake-sensor",
		Namespace: "fake",
	},
	Spec: v1alpha1.SensorSpec{
		Triggers: []v1alpha1.Trigger{
			{
				Template: &v1alpha1.TriggerTemplate{
					Name: "fake-trigger",
					GroupVersionResource: &metav1.GroupVersionResource{
						Group:    "apps",
						Version:  "v1",
						Resource: "deployments",
					},
				},
			},
		},
	},
}

func newUnstructured(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
				"labels": map[string]interface{}{
					"name": name,
				},
			},
		},
	}
}

func TestFetchResource(t *testing.T) {
	deployment := newUnstructured("apps/v1", "Deployment", "fake", "test")
	sensorObj.Spec.Triggers[0].Template.Source = &v1alpha1.ArtifactLocation{
		Resource: deployment,
	}
	uObj, err := FetchResource(fake.NewSimpleClientset(), sensorObj.Namespace, &sensorObj.Spec.Triggers[0])
	assert.Nil(t, err)
	assert.NotNil(t, uObj)
	assert.Equal(t, deployment.GetName(), uObj.GetName())
}

