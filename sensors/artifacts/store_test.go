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

package artifacts

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type FakeWorkflowArtifactReader struct{}

func (f *FakeWorkflowArtifactReader) Read() ([]byte, error) {
	return []byte(workflowv1alpha1), nil
}

func TestFetchArtifact(t *testing.T) {
	reader := &FakeWorkflowArtifactReader{}
	obj, err := FetchArtifact(reader)
	assert.Nil(t, err)
	assert.Equal(t, "argoproj.io/v1alpha1", obj.GetAPIVersion())
	assert.Equal(t, "Workflow", obj.GetKind())
}

func TestGetCredentials(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	mySecretCredentials := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "testing",
		},
		Data: map[string][]byte{"access": []byte("token"), "secret": []byte("value")},
	}
	_, err := fakeClient.CoreV1().Secrets("testing").Create(context.TODO(), mySecretCredentials, metav1.CreateOptions{})
	assert.Nil(t, err)

	// creds should be nil for unknown minio type
	unknownArtifact := &v1alpha1.ArtifactLocation{}
	creds, err := GetCredentials(unknownArtifact)
	assert.Nil(t, creds)
	assert.Nil(t, err)
}

func TestGetArtifactReader(t *testing.T) {
	// test unknown failure
	location := &v1alpha1.ArtifactLocation{}
	creds := &Credentials{
		accessKey: "access",
		secretKey: "secret",
	}
	_, err := GetArtifactReader(location, creds)
	assert.NotNil(t, err)
}

func TestDecodeSensor(t *testing.T) {
	b, err := ioutil.ReadFile("../../examples/sensors/multi-trigger-sensor.yaml")
	assert.Nil(t, err)
	_, err = decodeAndUnstructure(b)
	assert.Nil(t, err)
}

func TestDecodeWorkflow(t *testing.T) {
	_, err := decodeAndUnstructure([]byte(workflowv1alpha1))
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

func TestDecodeDeploymentv1(t *testing.T) {
	_, err := decodeAndUnstructure([]byte(deploymentv1))
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

func TestDecodeJobv1(t *testing.T) {
	_, err := decodeAndUnstructure([]byte(jobv1))
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

func TestDecodeUnsupported(t *testing.T) {
	_, err := decodeAndUnstructure([]byte(unsupportedType))
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

func TestDecodeUnknown(t *testing.T) {
	_, err := decodeAndUnstructure([]byte(unsupportedType))
	assert.Nil(t, err, "expected nil error but got", err)
}
