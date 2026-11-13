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

package redis

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestValidateRedisEventSource(t *testing.T) {
	listener := &EventListener{}

	err := listener.ValidateEventSource(context.Background())
	assert.Error(t, err)
	assert.Equal(t, "host address must be specified", err.Error())

	content, err := os.ReadFile(fmt.Sprintf("%s/%s", sources.EventSourceDir, "redis.yaml"))
	assert.Nil(t, err)

	var eventSource *v1alpha1.EventSource
	err = yaml.Unmarshal(content, &eventSource)
	assert.Nil(t, err)
	assert.NotNil(t, eventSource.Spec.Redis)

	for name, value := range eventSource.Spec.Redis {
		fmt.Println(name)
		l := &EventListener{
			RedisEventSource: value,
		}
		err := l.ValidateEventSource(context.Background())
		assert.NoError(t, err)
	}
}

func TestValidateURL(t *testing.T) {
	url := &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "redis"}, Key: "url"}
	tests := []struct {
		name    string
		source  v1alpha1.RedisEventSource
		wantErr string
	}{
		{name: "url only", source: v1alpha1.RedisEventSource{URL: url, Channels: []string{"ch"}}},
		{name: "url and host", source: v1alpha1.RedisEventSource{URL: url, HostAddress: "localhost:6379", Channels: []string{"ch"}}, wantErr: "url cannot be combined"},
		{name: "url and password", source: v1alpha1.RedisEventSource{URL: url, Password: url, Channels: []string{"ch"}}, wantErr: "url cannot be combined"},
		{name: "url and username", source: v1alpha1.RedisEventSource{URL: url, Username: "u", Channels: []string{"ch"}}, wantErr: "url cannot be combined"},
		{name: "url and db", source: v1alpha1.RedisEventSource{URL: url, DB: 1, Channels: []string{"ch"}}, wantErr: "url cannot be combined"},
		{name: "neither url nor host", source: v1alpha1.RedisEventSource{Channels: []string{"ch"}}, wantErr: "host address must be specified"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := tc.source
			err := validate(&source)
			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}
