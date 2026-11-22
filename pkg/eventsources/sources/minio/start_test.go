/*
Copyright 2018 The Argoproj Authors.

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

package minio

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

func TestNewClient(t *testing.T) {
	t.Run("returns a client for a valid endpoint", func(t *testing.T) {
		listener := &EventListener{
			EventName:        "example",
			MinioEventSource: v1alpha1.S3Artifact{Endpoint: "minio-service.argo-events:9000", Insecure: true},
		}

		client, err := listener.newClient("access", "secret", nil)
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("reports why the client could not be created", func(t *testing.T) {
		listener := &EventListener{
			EventName: "example",
			// The minio client rejects an endpoint that has a path.
			MinioEventSource: v1alpha1.S3Artifact{Endpoint: "minio-service.argo-events:9000/input", Insecure: true},
		}

		client, err := listener.newClient("access", "secret", nil)
		assert.Nil(t, client)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "failed to create a client for event source example, ")
			assert.NotContains(t, err.Error(), "%!w(<nil>)")
			assert.NotNil(t, errors.Unwrap(err), "the error from the minio client should be wrapped")
		}
	})
}
