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

package imap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

func TestEventListener_GetEventSourceName(t *testing.T) {
	listener := &EventListener{
		EventSourceName: "test-event-source",
		EventName:       "test-event",
	}
	assert.Equal(t, "test-event-source", listener.GetEventSourceName())
}

func TestEventListener_GetEventName(t *testing.T) {
	listener := &EventListener{
		EventSourceName: "test-event-source",
		EventName:       "test-event",
	}
	assert.Equal(t, "test-event", listener.GetEventName())
}

func TestEventListener_GetEventSourceType(t *testing.T) {
	listener := &EventListener{}
	assert.Equal(t, v1alpha1.IMAPEvent, listener.GetEventSourceType())
}

func TestValidateEventSource(t *testing.T) {
	tests := []struct {
		name        string
		eventSource *v1alpha1.IMAPEventSource
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "nil event source",
			eventSource: nil,
			wantErr:     true,
			errMessage:  "invalid event source: nil",
		},
		{
			name: "missing hostAddress",
			eventSource: &v1alpha1.IMAPEventSource{
				HostAddress: "",
				Username: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "username",
				},
				Password: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "password",
				},
			},
			wantErr:    true,
			errMessage: "hostAddress must be specified",
		},
		{
			name: "missing username",
			eventSource: &v1alpha1.IMAPEventSource{
				HostAddress: "mail.example.com:993",
				Password: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "password",
				},
			},
			wantErr:    true,
			errMessage: "username must be specified",
		},
		{
			name: "missing password",
			eventSource: &v1alpha1.IMAPEventSource{
				HostAddress: "mail.example.com:993",
				Username: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "username",
				},
			},
			wantErr:    true,
			errMessage: "password must be specified",
		},
		{
			name: "valid config with all required fields",
			eventSource: &v1alpha1.IMAPEventSource{
				HostAddress: "mail.example.com:993",
				Username: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "username",
				},
				Password: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "password",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with TLS",
			eventSource: &v1alpha1.IMAPEventSource{
				HostAddress: "mail.example.com:993",
				Username: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "username",
				},
				Password: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "password",
				},
				StartTLS: true,
			},
			wantErr: false,
		},
		{
			name: "valid config with all fields",
			eventSource: &v1alpha1.IMAPEventSource{
				HostAddress: "mail.example.com:993",
				Username: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "username",
				},
				Password: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "password",
				},
				StartTLS: true,
				Metadata: map[string]string{
					"source": "test",
					"env":    "production",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.eventSource == nil {
				listener := &EventListener{
					IMAPEventSource: v1alpha1.IMAPEventSource{},
				}
				err := listener.ValidateEventSource(context.Background())
				assert.Error(t, err)
			} else {
				listener := &EventListener{
					IMAPEventSource: *tt.eventSource,
				}
				err := listener.ValidateEventSource(context.Background())
				if tt.wantErr {
					assert.Error(t, err)
					if tt.errMessage != "" {
						assert.Equal(t, tt.errMessage, err.Error())
					}
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}
