/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package store

import (
	"io/ioutil"
	"k8s.io/client-go/kubernetes/fake"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FakeWorkflowArtifactReader struct{}

func (f *FakeWorkflowArtifactReader) Read() ([]byte, error) {
	return []byte(workflowv1alpha1), nil
}

func TestFetchArtifact(t *testing.T) {
	reader := &FakeWorkflowArtifactReader{}
	gvk := &metav1.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Workflow",
	}
	obj, err := FetchArtifact(reader, gvk)
	assert.Nil(t, err)
	assert.Equal(t, "argoproj.io/v1alpha1", obj.GetAPIVersion())
	assert.Equal(t, "Workflow", obj.GetKind())
}

func TestGetArtifactReader(t *testing.T) {
	// test unknown failure
	location := &v1alpha1.ArtifactLocation{}
	creds := &Credentials{
		accessKey: "access",
		secretKey: "secret",
	}
	clientset := fake.NewSimpleClientset()
	_, err := GetArtifactReader(location, creds, clientset)
	assert.NotNil(t, err)
}

func TestDecodeAndUnstructure(t *testing.T) {
	t.Run("sensor", decodeSensor)
	t.Run("workflow", decodeWorkflow)
	// Note that since #16 - Restrict ResourceObject creation via RBAC roles
	// decoding&converting to unstructure objects should pass fine for any valid objects
	// the store no longer should control restrictions around object creation
	t.Run("unsupported", decodeUnsupported)
	t.Run("unknown", decodeUnknown)
}

func decodeSensor(t *testing.T) {
	b, err := ioutil.ReadFile("../examples/sensors/multi-trigger-sensor.yaml")
	assert.Nil(t, err)

	gvk := &metav1.GroupVersionKind{
		Group:   v1alpha1.SchemaGroupVersionKind.Group,
		Version: v1alpha1.SchemaGroupVersionKind.Version,
		Kind:    v1alpha1.SchemaGroupVersionKind.Kind,
	}

	_, err = decodeAndUnstructure(b, gvk)
	assert.Nil(t, err)
}

func decodeWorkflow(t *testing.T) {
	gvk := &metav1.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    "Workflow",
	}
	_, err := decodeAndUnstructure([]byte(workflowv1alpha1), gvk)
	assert.Nil(t, err)
}

var workflowv1alpha1 = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

func decodeDeploymentv1(t *testing.T) {
	gvk := &metav1.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}
	_, err := decodeAndUnstructure([]byte(deploymentv1), gvk)
	assert.Nil(t, err)
}

var deploymentv1 = `
{
	"apiVersion": "apps/v1",
	"kind": "Deployment",
	"metadata": {
	  "name": "nginx-deployment",
	  "labels": {
		"app": "nginx"
	  }
	},
	"spec": {
	  "replicas": 3,
	  "selector": {
		"matchLabels": {
		  "app": "nginx"
		}
	  },
	  "template": {
		"metadata": {
		  "labels": {
			"app": "nginx"
		  }
		},
		"spec": {
		  "containers": [
			{
			  "name": "nginx",
			  "image": "nginx:1.7.9",
			  "ports": [
				{
				  "containerPort": 80
				}
			  ]
			}
		  ]
		}
	  }
	}
  }
`

func decodeJobv1(t *testing.T) {
	gvk := &metav1.GroupVersionKind{
		Group:   "batch",
		Version: "v1",
		Kind:    "Job",
	}
	_, err := decodeAndUnstructure([]byte(jobv1), gvk)
	assert.Nil(t, err)
}

var jobv1 = `
apiVersion: batch/v1
kind: Job
metadata:
  # Unique key of the Job instance
  name: example-job
spec:
  template:
    metadata:
      name: example-job
    spec:
      containers:
      - name: pi
        image: perl
        command: ["perl"]
        args: ["-Mbignum=bpi", "-wle", "print bpi(2000)"]
      # Do not restart containers after they exit
      restartPolicy: Never
`

func decodeUnsupported(t *testing.T) {
	gvk := &metav1.GroupVersionKind{
		Group:   "batch",
		Version: "v1",
		Kind:    "Job",
	}
	_, err := decodeAndUnstructure([]byte(unsupportedType), gvk)
	assert.Nil(t, err)
}

var unsupportedType = `
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  # Unique key of the DaemonSet instance
  name: daemonset-example
spec:
  template:
    metadata:
      labels:
        app: daemonset-example
    spec:
      containers:
      # This container is run once on each Node in the cluster
      - name: daemonset-example
        image: ubuntu:trusty
        command:
        - /bin/sh
        args:
        - -c
        - >-
          while [ true ]; do
          echo "DaemonSet running on $(hostname)" ;
          sleep 10 ;
          done
`

func decodeUnknown(t *testing.T) {
	gvk := &metav1.GroupVersionKind{
		Group:   "unknown",
		Version: "123",
		Kind:    "What??",
	}
	_, err := decodeAndUnstructure([]byte(unsupportedType), gvk)
	assert.Nil(t, err, "expected nil error but got", err)
}
