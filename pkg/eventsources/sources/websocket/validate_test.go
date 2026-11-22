/*
Copyright 2026 The Argoproj Authors.

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

package websocket

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

func TestValidateEventSource(t *testing.T) {
	listener := &EventListener{}

	t.Run("nil event source", func(t *testing.T) {
		err := validate(nil)
		assert.ErrorIs(t, err, v1alpha1.ErrNilEventSource)
	})

	t.Run("missing url", func(t *testing.T) {
		err := validate(&v1alpha1.WebSocketEventSource{})
		assert.EqualError(t, err, "url must be specified")
	})

	t.Run("invalid url scheme", func(t *testing.T) {
		err := validate(&v1alpha1.WebSocketEventSource{URL: "http://example.com/socket"})
		assert.EqualError(t, err, `url must start with ws:// or wss://, got "http://example.com/socket"`)
	})

	t.Run("valid ws url", func(t *testing.T) {
		err := validate(&v1alpha1.WebSocketEventSource{URL: "ws://example.com/socket"})
		assert.NoError(t, err)
	})

	t.Run("valid wss url", func(t *testing.T) {
		err := validate(&v1alpha1.WebSocketEventSource{URL: "wss://example.com/socket"})
		assert.NoError(t, err)
	})

	t.Run("invalid basic auth", func(t *testing.T) {
		err := validate(&v1alpha1.WebSocketEventSource{
			URL:  "wss://example.com/socket",
			Auth: &v1alpha1.BasicAuth{Username: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "s"}, Key: "u"}},
		})
		assert.Error(t, err)
	})

	t.Run("listener validate", func(t *testing.T) {
		listener.WebSocketEventSource = v1alpha1.WebSocketEventSource{URL: "ws://example.com/socket"}
		assert.NoError(t, listener.ValidateEventSource(context.Background()))
	})
}
