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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestGetCredentials(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	mySecretCredentials := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "testing",
		},
		Data: map[string][]byte{"access": []byte("token"), "secret": []byte("value")},
	}
	_, err := fakeClient.CoreV1().Secrets("testing").Create(context.Background(), mySecretCredentials, metav1.CreateOptions{})
	assert.Nil(t, err)

	// creds should be nil for unknown minio type
	unknownArtifact := &v1alpha1.ArtifactLocation{}
	creds, err := GetCredentials(unknownArtifact)
	assert.Nil(t, creds)
	assert.Nil(t, err)
}
