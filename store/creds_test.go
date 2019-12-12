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
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
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
	_, err := fakeClient.CoreV1().Secrets("testing").Create(mySecretCredentials)
	assert.Nil(t, err)

	// creds should be nil for unknown minio type
	unknownArtifact := &v1alpha1.ArtifactLocation{}
	creds, err := GetCredentials(fakeClient, "testing", unknownArtifact)
	assert.Nil(t, creds)
	assert.Nil(t, err)

	// succeed for S3 minio type
	s3Artifact := &v1alpha1.ArtifactLocation{
		S3: &apicommon.S3Artifact{
			AccessKey: &apiv1.SecretKeySelector{
				LocalObjectReference: apiv1.LocalObjectReference{Name: "test"},
				Key:                  "access",
			},
			SecretKey: &apiv1.SecretKeySelector{
				LocalObjectReference: apiv1.LocalObjectReference{Name: "test"},
				Key:                  "secret",
			},
			Bucket: &apicommon.S3Bucket{
				Name: "test-bucket",
			},
		},
	}
	creds, err = GetCredentials(fakeClient, "testing", s3Artifact)
	assert.Nil(t, err)
	assert.NotNil(t, creds)
	assert.Equal(t, "token", creds.accessKey)
	assert.Equal(t, "value", creds.secretKey)
}

func TestGetSecrets(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	mySecretCredentials := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "testing",
		},
		Data: map[string][]byte{"access": []byte("token"), "secret": []byte("value")},
	}
	_, err := fakeClient.CoreV1().Secrets("testing").Create(mySecretCredentials)
	assert.Nil(t, err)

	// get valid secret with present key
	pValue, err := GetSecrets(fakeClient, "testing", "test", "access")
	assert.Nil(t, err)
	assert.Equal(t, "token", pValue)

	// get valid secret with non-present key
	_, err = GetSecrets(fakeClient, "testing", "test", "unknown")
	assert.NotNil(t, err)

	// get invalid secret
	_, err = GetSecrets(fakeClient, "testing", "unknown", "access")
	assert.NotNil(t, err)
}
