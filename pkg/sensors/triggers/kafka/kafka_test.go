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
package kafka

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
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
						URL:            "fake-kafka-url",
						Topic:          "fake-topic",
						Parameters:     nil,
						RequiredAcks:   1,
						Compress:       false,
						FlushFrequency: 0,
						SASL: &v1alpha1.SASLConfig{
							Mechanism: "PLAIN",
						},
						TLS:             nil,
						Payload:         nil,
						PartitioningKey: nil,
					},
				},
			},
		},
	},
}

func getFakeKafkaTrigger(producers sharedutil.StringKeyedMap[sarama.AsyncProducer]) (*KafkaTrigger, error) {
	return NewKafkaTrigger(sensorObj.DeepCopy(), sensorObj.Spec.Triggers[0].DeepCopy(), producers, logging.NewArgoEventsLogger())
}

func TestNewKafkaTrigger(t *testing.T) {
	producer := mocks.NewAsyncProducer(t, nil)
	producers := sharedutil.NewStringKeyedMap[sarama.AsyncProducer]()
	producers.Store("fake-trigger", producer)
	trigger, err := NewKafkaTrigger(sensorObj.DeepCopy(), sensorObj.Spec.Triggers[0].DeepCopy(), producers, logging.NewArgoEventsLogger())

	assert.Nil(t, err)
	assert.Equal(t, trigger.Trigger.Template.Kafka.URL, "fake-kafka-url")
	assert.Equal(t, trigger.Trigger.Template.Kafka.SASL.Mechanism, "PLAIN")
}

func TestKafkaTrigger_FetchResource(t *testing.T) {
	producer := mocks.NewAsyncProducer(t, nil)
	producers := sharedutil.NewStringKeyedMap[sarama.AsyncProducer]()
	producers.Store("fake-trigger", producer)
	trigger, err := getFakeKafkaTrigger(producers)
	assert.Nil(t, err)
	obj, err := trigger.FetchResource(context.TODO())
	assert.Nil(t, err)
	assert.NotNil(t, obj)
	trigger1, ok := obj.(*v1alpha1.KafkaTrigger)
	assert.Equal(t, true, ok)
	assert.Equal(t, trigger.Trigger.Template.Kafka.URL, trigger1.URL)
}

func TestKafkaTrigger_ApplyResourceParameters(t *testing.T) {
	producer := mocks.NewAsyncProducer(t, nil)
	producers := sharedutil.NewStringKeyedMap[sarama.AsyncProducer]()
	producers.Store("fake-trigger", producer)
	trigger, err := getFakeKafkaTrigger(producers)
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
			Data: []byte(`{"url": "another-fake-kafka-url"}`),
		},
	}

	defaultValue := "http://default.com"
	secureHeader := &v1alpha1.SecureHeader{Name: "test", ValueFrom: &v1alpha1.ValueFromSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: "tokens",
			},
			Key: "serviceToken"}},
	}

	secureHeaders := []*v1alpha1.SecureHeader{}
	secureHeaders = append(secureHeaders, secureHeader)
	trigger.Trigger.Template.Kafka.Headers = map[string]string{"key": "value"}
	trigger.Trigger.Template.Kafka.SecureHeaders = secureHeaders
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

	resource, err := trigger.ApplyResourceParameters(testEvents, trigger.Trigger.Template.Kafka)
	assert.Nil(t, err)
	assert.NotNil(t, resource)

	updatedTrigger, ok := resource.(*v1alpha1.KafkaTrigger)
	assert.Equal(t, "value", updatedTrigger.Headers["key"])
	assert.Equal(t, "serviceToken", updatedTrigger.SecureHeaders[0].ValueFrom.SecretKeyRef.Key)
	assert.Nil(t, err)
	assert.Equal(t, true, ok)
	assert.Equal(t, "another-fake-kafka-url", updatedTrigger.URL)
}

func TestKafkaTrigger_Execute(t *testing.T) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer := mocks.NewAsyncProducer(t, config)
	producers := sharedutil.NewStringKeyedMap[sarama.AsyncProducer]()
	producers.Store("fake-trigger", producer)
	trigger, err := getFakeKafkaTrigger(producers)
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
	trigger.Trigger.Template.Kafka.Headers = map[string]string{"key1": "value1", "key2": "value2"}
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

	result, err := trigger.Execute(context.TODO(), testEvents, trigger.Trigger.Template.Kafka)
	assert.Nil(t, err)
	assert.Nil(t, result)

	select {
	case kafkaMessage := <-producer.Successes():
		assert.NotNil(t, kafkaMessage)
		assert.Equal(t, 2, len(kafkaMessage.Headers))

		// Convert to map for order-independent comparison
		headerMap := make(map[string]string)
		for _, h := range kafkaMessage.Headers {
			headerMap[string(h.Key)] = string(h.Value)
		}

		assert.Equal(t, "value1", headerMap["key1"])
		assert.Equal(t, "value2", headerMap["key2"])
	case <-time.After(1 * time.Second):
		assert.Fail(t, "timed out waiting for message to contain headers")
	}
}
