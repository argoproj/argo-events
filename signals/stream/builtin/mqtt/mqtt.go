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

package mqtt

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk"
	MQTTlib "github.com/eclipse/paho.mqtt.golang"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	topicKey  = "topic"
	EventType = "mqtt.github.io.msg"
)

// Note: micro requires stateless operation so the Listen() method should not use the
// receive struct to save or modify state.
type mqtt struct{}

// New creates a new mqtt listener
func New() sdk.Listener {
	return new(mqtt)
}

func (*mqtt) Listen(signal *v1alpha1.Signal, done <-chan struct{}) (<-chan *v1alpha1.Event, error) {
	// parse out the attributes
	topic, ok := signal.Stream.Attributes[topicKey]
	if !ok {
		return nil, sdk.ErrMissingRequiredAttribute
	}

	events := make(chan *v1alpha1.Event)

	handler := func(c MQTTlib.Client, msg MQTTlib.Message) {
		event := &v1alpha1.Event{
			Context: v1alpha1.EventContext{
				EventID:            strconv.FormatUint(uint64(msg.MessageID()), 10),
				EventType:          EventType,
				CloudEventsVersion: sdk.CloudEventsVersion,
				EventTime:          metav1.Time{Time: time.Now().UTC()},
				Extensions:         make(map[string]string),
			},
			Data: msg.Payload(),
		}
		log.Printf("signal '%s' received msg", signal.Name)
		events <- event
	}

	opts := MQTTlib.NewClientOptions().AddBroker(signal.Stream.URL).SetClientID(signal.Name)
	client := MQTTlib.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to client: %s", token.Error())
	}

	if token := client.Subscribe(topic, 0, handler); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to subscribe to topic: %s", token.Error())
	}

	// wait for done signal
	go func() {
		defer close(events)
		<-done
		if token := client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
			log.Printf("failed to unsubscribe from topic: %s", token.Error())
		}
		client.Disconnect(0)
		log.Printf("shut down signal '%s'", signal.Name)
	}()

	log.Printf("signal '%s' listening for mqtt msgs on topic [%s]...", signal.Name, topic)
	return events, nil
}
