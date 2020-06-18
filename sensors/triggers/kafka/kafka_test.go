/*
Copyright 2020 BlackRock, Inc.

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
package kafka

import (
	"testing"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
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
					Kafka: &v1alpha1.KafkaTrigger{
						URL:             "fake-kafka-url",
						Topic:           "fake-topic",
						Partition:       0,
						Parameters:      nil,
						RequiredAcks:    1,
						Compress:        false,
						FlushFrequency:  0,
						TLS:             nil,
						Payload:         nil,
						PartitioningKey: "",
					},
				},
			},
		},
	},
}

func getFakeKafkaTrigger(producers map[string]sarama.AsyncProducer) (*KafkaTrigger, error) {
	return NewKafkaTrigger(sensorObj.DeepCopy(), sensorObj.Spec.Triggers[0].DeepCopy(), producers, common.NewArgoEventsLogger())
}

func TestKafkaTrigger_FetchResource(t *testing.T) {
	producer := mocks.NewAsyncProducer(t, nil)
	trigger, err := getFakeKafkaTrigger(map[string]sarama.AsyncProducer{
		"fake-trigger": producer,
	})
	assert.Nil(t, err)
	obj, err := trigger.FetchResource()
	assert.Nil(t, err)
	assert.NotNil(t, obj)
	trigger1, ok := obj.(*v1alpha1.KafkaTrigger)
	assert.Equal(t, true, ok)
	assert.Equal(t, trigger.Trigger.Template.Kafka.URL, trigger1.URL)
}

func TestKafkaTrigger_ApplyResourceParameters(t *testing.T) {
	producer := mocks.NewAsyncProducer(t, nil)
	trigger, err := getFakeKafkaTrigger(map[string]sarama.AsyncProducer{
		"fake-trigger": producer,
	})
	assert.Nil(t, err)
	id := trigger.Sensor.NodeID("fake-dependency")
	trigger.Sensor.Status = v1alpha1.SensorStatus{
		Nodes: map[string]v1alpha1.NodeStatus{
			id: {
				Name: "fake-dependency",
				Type: v1alpha1.NodeTypeEventDependency,
				ID:   id,
				Event: &v1alpha1.Event{
					Context: &v1alpha1.EventContext{
						ID:              "1",
						Type:            "webhook",
						Source:          "webhook-gateway",
						DataContentType: "application/json",
						SpecVersion:     cloudevents.VersionV1,
						Subject:         "example-1",
					},
					Data: []byte(`{"url": "another-fake-kafka-url"}`),
				},
			},
		},
	}

	defaultValue := "http://default.com"

	trigger.Trigger.Template.Kafka.Parameters = []v1alpha1.TriggerParameter{
		{
			Src: &v1alpha1.TriggerParameterSource{
				DependencyName: "fake-dependency",
				DataKey:        "url",
				Value:          &defaultValue,
			},
			Dest: "url",
		},
	}

	resource, err := trigger.ApplyResourceParameters(trigger.Sensor, trigger.Trigger.Template.Kafka)
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	updatedTrigger, ok := resource.(*v1alpha1.KafkaTrigger)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)
	assert.Equal(t, "another-fake-kafka-url", updatedTrigger.URL)
}

func TestKafkaTrigger_Execute(t *testing.T) {
	producer := mocks.NewAsyncProducer(t, nil)
	trigger, err := getFakeKafkaTrigger(map[string]sarama.AsyncProducer{
		"fake-trigger": producer,
	})
	assert.Nil(t, err)
	id := trigger.Sensor.NodeID("fake-dependency")
	trigger.Sensor.Status = v1alpha1.SensorStatus{
		Nodes: map[string]v1alpha1.NodeStatus{
			id: {
				Name: "fake-dependency",
				Type: v1alpha1.NodeTypeEventDependency,
				ID:   id,
				Event: &v1alpha1.Event{
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
			},
		},
	}
	defaultValue := "hello"

	trigger.Trigger.Template.Kafka.Payload = []v1alpha1.TriggerParameter{
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

	result, err := trigger.Execute(trigger.Trigger.Template.Kafka)
	assert.Nil(t, err)
	assert.Nil(t, result)
}
