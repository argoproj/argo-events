/*
Copyright 2021 BlackRock, Inc.

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
package pulsar

import (
	"context"
	"testing"

	"github.com/apache/pulsar-client-go/pulsar"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

type mockPulsarProducer struct {
	topic    string
	name     string
	expected bool
}

func (m *mockPulsarProducer) ExpectInputAndSucceed() {
	m.expected = true
}
func (m *mockPulsarProducer) Topic() string {
	return m.topic
}
func (m *mockPulsarProducer) Name() string {
	return m.name
}
func (m *mockPulsarProducer) Send(context.Context, *pulsar.ProducerMessage) (pulsar.MessageID, error) {
	if m.expected {
		m.expected = false
		return nil, nil
	}
	return nil, errors.New("input not expected")
}
func (m *mockPulsarProducer) SendAsync(context.Context, *pulsar.ProducerMessage, func(pulsar.MessageID, *pulsar.ProducerMessage, error)) {

}
func (m *mockPulsarProducer) LastSequenceID() int64 {
	return 0
}
func (m *mockPulsarProducer) Flush() error {
	return nil
}
func (m *mockPulsarProducer) Close() {

}

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
					Pulsar: &v1alpha1.PulsarTrigger{
						URL:        "fake-pulsar-url",
						Topic:      "fake-topic",
						Parameters: nil,
						Payload:    nil,
					},
				},
			},
		},
	},
}

func getFakePulsarTrigger(producers map[string]pulsar.Producer) (*PulsarTrigger, error) {
	return NewPulsarTrigger(sensorObj.DeepCopy(), sensorObj.Spec.Triggers[0].DeepCopy(), producers, logging.NewArgoEventsLogger())
}

func TestNewPulsarTrigger(t *testing.T) {
	producer := &mockPulsarProducer{
		topic: "fake-topic",
		name:  "fake-producer",
	}
	producers := map[string]pulsar.Producer{
		"fake-trigger": producer,
	}
	trigger, err := NewPulsarTrigger(sensorObj.DeepCopy(), sensorObj.Spec.Triggers[0].DeepCopy(), producers, logging.NewArgoEventsLogger())
	assert.Nil(t, err)
	assert.Equal(t, trigger.Trigger.Template.Pulsar.URL, "fake-pulsar-url")
	assert.Equal(t, trigger.Trigger.Template.Pulsar.Topic, "fake-topic")
}

func TestPulsarTrigger_FetchResource(t *testing.T) {
	producer := &mockPulsarProducer{
		topic: "fake-topic",
		name:  "fake-producer",
	}
	trigger, err := getFakePulsarTrigger(map[string]pulsar.Producer{
		"fake-trigger": producer,
	})
	assert.Nil(t, err)
	obj, err := trigger.FetchResource(context.TODO())
	assert.Nil(t, err)
	assert.NotNil(t, obj)
	trigger1, ok := obj.(*v1alpha1.PulsarTrigger)
	assert.Equal(t, true, ok)
	assert.Equal(t, trigger.Trigger.Template.Pulsar.URL, trigger1.URL)
}

func TestPulsarTrigger_ApplyResourceParameters(t *testing.T) {
	producer := &mockPulsarProducer{
		topic: "fake-topic",
		name:  "fake-producer",
	}
	trigger, err := getFakePulsarTrigger(map[string]pulsar.Producer{
		"fake-trigger": producer,
	})
	assert.Nil(t, err)

	testEvents := map[string]*v1alpha1.Event{
		"fake-dependency": {
			Context: &v1alpha1.EventContext{
				ID:              "1",
				Type:            "webhook",
				Source:          "webhook-gateway",
				DataContentType: "application/json",
				SpecVersion:     cloudevents.VersionV1,
				Subject:         "example-1",
			},
			Data: []byte(`{"url": "another-fake-pulsar-url"}`),
		},
	}

	defaultValue := "http://default.com"

	trigger.Trigger.Template.Pulsar.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "url",
				Value:          &defaultValue,
			},
			Dest: "url",
		},
	}

	resource, err := trigger.ApplyResourceParameters(testEvents, trigger.Trigger.Template.Pulsar)
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	updatedTrigger, ok := resource.(*v1alpha1.PulsarTrigger)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)
	assert.Equal(t, "another-fake-pulsar-url", updatedTrigger.URL)
}

func TestPulsarTrigger_Execute(t *testing.T) {
	producer := &mockPulsarProducer{
		topic: "fake-topic",
		name:  "fake-producer",
	}
	trigger, err := getFakePulsarTrigger(map[string]pulsar.Producer{
		"fake-trigger": producer,
	})
	assert.Nil(t, err)

	testEvents := map[string]*v1alpha1.Event{
		"fake-dependency": {
			Context: &v1alpha1.EventContext{
				ID:              "1",
				Type:            "webhook",
				Source:          "webhook-gateway",
				DataContentType: "application/json",
				SpecVersion:     cloudevents.VersionV1,
				Subject:         "example-1",
			},
			Data: []byte(`{"message": "world"}`),
		},
	}

	defaultValue := "hello"

	trigger.Trigger.Template.Pulsar.Payload = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "message",
				Value:          &defaultValue,
			},
			Dest: "message",
		},
	}

	producer.ExpectInputAndSucceed()

	result, err := trigger.Execute(context.TODO(), testEvents, trigger.Trigger.Template.Pulsar)
	assert.Nil(t, err)
	assert.Nil(t, result)
}
