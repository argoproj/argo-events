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

package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

func TestValidateWebhookContextBasicAuth(t *testing.T) {
	t.Run("valid basicAuth", func(t *testing.T) {
		context := &aev1.WebhookContext{
			Endpoint: "/fake",
			Port:     "12000",
			BasicAuth: &aev1.BasicAuth{
				Username: &corev1.SecretKeySelector{Key: "username"},
				Password: &corev1.SecretKeySelector{Key: "password"},
			},
		}
		assert.NoError(t, ValidateWebhookContext(context))
	})

	t.Run("basicAuth missing password", func(t *testing.T) {
		context := &aev1.WebhookContext{
			Endpoint: "/fake",
			Port:     "12000",
			BasicAuth: &aev1.BasicAuth{
				Username: &corev1.SecretKeySelector{Key: "username"},
			},
		}
		assert.Error(t, ValidateWebhookContext(context))
	})

	t.Run("authSecret and basicAuth are mutually exclusive", func(t *testing.T) {
		context := &aev1.WebhookContext{
			Endpoint:   "/fake",
			Port:       "12000",
			AuthSecret: &corev1.SecretKeySelector{Key: "token"},
			BasicAuth: &aev1.BasicAuth{
				Username: &corev1.SecretKeySelector{Key: "username"},
				Password: &corev1.SecretKeySelector{Key: "password"},
			},
		}
		assert.Error(t, ValidateWebhookContext(context))
	})
}
