/*
Copyright 2020 The Argoproj Authors.

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
package email

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	"github.com/argoproj/notifications-engine/pkg/services"
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
					Email: &v1alpha1.EmailTrigger{
						SMTPPassword: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "secret",
							},
							Key: "password",
						},
						Host:     "fake-host",
						Port:     468,
						Username: "fake-username",
						To:       []string{"fake1@email.com", "fake2@email.com"},
						From:     "fake-email",
						Subject:  "fake-subject",
						Body:     "fake-body",
					},
				},
			},
		},
	},
}

func getEmailTrigger(n services.NotificationService) *EmailTrigger {
	return &EmailTrigger{
		Sensor:   sensorObj.DeepCopy(),
		Trigger:  sensorObj.Spec.Triggers[0].DeepCopy(),
		Logger:   logging.NewArgoEventsLogger(),
		emailSvc: n,
	}
}

func TestEmailTrigger_FetchResource(t *testing.T) {
	trigger := getEmailTrigger(nil)
	resource, err := trigger.FetchResource(context.TODO())
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	trigger, ok := resource.(*v1alpha1.EmailTrigger)
	assert.Equal(t, true, ok)
	assert.Equal(t, "fake-host", trigger.Host)
	assert.Equal(t, int32(468), trigger.Port)
	assert.Equal(t, "fake-username", trigger.Username)
	assert.Equal(t, []string{"fake1@email.com", "fake2@email.com"}, trigger.To)
	assert.Equal(t, "fake-email", trigger.From)
	assert.Equal(t, "fake-subject", trigger.Subject)
	assert.Equal(t, "fake-body", trigger.Body)
}

func TestEmailTrigger_ApplyResourceParameters(t *testing.T) {
	trigger := getEmailTrigger(nil)

	testEvents := map[string]*v1alpha1.Event{
		"fake-dependency": {
			Context: &v1alpha1.EventContext{
				ID:              "1",
				Type:            "webhook",
				Source:          "webhook-gateway",
				DataContentType: "application/json",
				SpecVersion:     "1.0",
				Subject:         "example-1",
			},
			Data: []byte(`{"to": "real@email.com", "name": "Luke"}`),
		},
	}

	trigger.Trigger.Template.Email.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "to",
			},
			Dest: "to.0",
		},
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "body",
				DataTemplate:   "Hi {{.Input.name}},\n\tHello There.\nThanks,\nObi",
			},
			Dest: "body",
		},
	}

	resource, err := trigger.ApplyResourceParameters(testEvents, trigger.Trigger.Template.Email)
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	trigger, ok := resource.(*v1alpha1.EmailTrigger)
	assert.Equal(t, true, ok)
	assert.Equal(t, "fake-host", trigger.Host)
	assert.Equal(t, int32(468), trigger.Port)
	assert.Equal(t, "fake-username", trigger.Username)
	assert.Equal(t, []string{"real@email.com", "fake2@email.com"}, trigger.To)
	assert.Equal(t, "fake-email", trigger.From)
	assert.Equal(t, "fake-subject", trigger.Subject)
	assert.Equal(t, "Hi Luke,\n\tHello There.\nThanks,\nObi", trigger.Body)
}

// Mock Notification Service that returns an error on Send
type MockNotificationServiceError struct{}

// Mocks a send error
func (m *MockNotificationServiceError) Send(n services.Notification, d services.Destination) error {
	return errors.New("")
}

// Mock Notification Service that returns nil on Send
type MockNotificationService struct{}

// Mocks a successful send
func (m *MockNotificationService) Send(n services.Notification, d services.Destination) error {
	return nil
}

func TestEmailTrigger_Execute(t *testing.T) {
	t.Run("Unmarshallable resource", func(t *testing.T) {
		trigger := getEmailTrigger(&MockNotificationService{})
		_, err := trigger.Execute(context.TODO(), map[string]*v1alpha1.Event{}, nil)
		assert.NotNil(t, err)
	})

	t.Run("Empty to scenario", func(t *testing.T) {
		trigger := getEmailTrigger(&MockNotificationService{})
		trigger.Trigger.Template.Email.To = make([]string, 0)
		_, err := trigger.Execute(context.TODO(), map[string]*v1alpha1.Event{}, trigger.Trigger.Template.Email)
		assert.NotNil(t, err)
	})

	t.Run("Invalid to scenario", func(t *testing.T) {
		trigger := getEmailTrigger(&MockNotificationService{})
		trigger.Trigger.Template.Email.To = []string{"not@a@valid.email"}
		_, err := trigger.Execute(context.TODO(), map[string]*v1alpha1.Event{}, trigger.Trigger.Template.Email)
		assert.NotNil(t, err)
	})

	t.Run("Empty subject scenario", func(t *testing.T) {
		trigger := getEmailTrigger(&MockNotificationService{})
		trigger.Trigger.Template.Email.Subject = ""
		_, err := trigger.Execute(context.TODO(), map[string]*v1alpha1.Event{}, trigger.Trigger.Template.Email)
		assert.NotNil(t, err)
	})

	t.Run("Empty body scenario", func(t *testing.T) {
		trigger := getEmailTrigger(&MockNotificationService{})
		trigger.Trigger.Template.Email.Body = ""
		_, err := trigger.Execute(context.TODO(), map[string]*v1alpha1.Event{}, trigger.Trigger.Template.Email)
		assert.NotNil(t, err)
	})

	t.Run("Error when sending email", func(t *testing.T) {
		trigger := getEmailTrigger(&MockNotificationServiceError{})
		_, err := trigger.Execute(context.TODO(), map[string]*v1alpha1.Event{}, trigger.Trigger.Template.Email)
		assert.NotNil(t, err)
	})

	t.Run("Email send successfully", func(t *testing.T) {
		trigger := getEmailTrigger(&MockNotificationService{})
		_, err := trigger.Execute(context.TODO(), map[string]*v1alpha1.Event{}, trigger.Trigger.Template.Email)
		assert.Nil(t, err)
	})
}
