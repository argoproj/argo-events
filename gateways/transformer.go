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

package gateways

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/argoproj/argo-events/common"
	sv1alpha "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	suuid "github.com/satori/go.uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TransformerPayload contains payload of cloudevents.
type TransformerPayload struct {
	// Src contains information about which specific configuration in gateway generated the event
	Src string `json:"src"`
	// Payload is event data
	Payload []byte `json:"payload"`
}

// transformEvent transforms an event from event source into a CloudEvents specification compliant event
// See https://github.com/cloudevents/spec for more info.
func (gc *GatewayConfig) transformEvent(gatewayEvent *Event) (*sv1alpha.Event, error) {
	// Generate an event id
	eventId := suuid.NewV1()

	gc.Log.Info().Str("source", *gatewayEvent.Name).
		Msg("converting gateway event into cloudevents specification compliant event")

	// Create an CloudEvent
	ce := &sv1alpha.Event{
		Context: sv1alpha.EventContext{
			CloudEventsVersion: common.CloudEventsVersion,
			EventID:            fmt.Sprintf("%x", eventId),
			ContentType:        "application/json",
			EventTime:          metav1.MicroTime{Time: time.Now().UTC()},
			EventType:          gc.gw.Spec.Type,
			EventTypeVersion:   gc.gw.Spec.EventVersion,
			Source: &sv1alpha.URI{
				Host: common.DefaultGatewayConfigurationName(gc.gw.Name, *gatewayEvent.Name),
			},
		},
		Payload: gatewayEvent.Payload,
	}

	gc.Log.Info().Str("event-source", *gatewayEvent.Name).Msg("event has been transformed into cloud event")
	return ce, nil
}

// dispatchEventOverHttp dispatches event to watchers over http.
func (gc *GatewayConfig) dispatchEventOverHttp(event *sv1alpha.Event) error {
	gc.Log.Info().Str("source", event.Context.Source.Host).Msg("dispatching event to watchers")

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to dispatch event to watchers over http. marshalling failed. err: %+v", err)
	}
	for _, sensor := range gc.gw.Spec.Watchers.Sensors {
		err := gc.postCloudEventToWatcher(common.DefaultSensorServiceName(sensor.Name), common.SensorServicePort, common.SensorServiceEndpoint, payload)
		if err != nil {
			return fmt.Errorf("failed to dispatch event to sensor watcher over http. communication error. err: %+v", err)
		}
	}
	for _, gateway := range gc.gw.Spec.Watchers.Gateways {
		err := gc.postCloudEventToWatcher(common.DefaultServiceName(gateway.Name), gateway.Port, gateway.Endpoint, payload)
		if err != nil {
			return fmt.Errorf("failed to dispatch event to gateway watcher over http. communication error. err: %+v", err)
		}
	}
	gc.Log.Info().Msg("successfully dispatched event to all watchers")
	return nil
}

// postCloudEventToWatcher makes a HTTP POST call to watcher's service
func (gc *GatewayConfig) postCloudEventToWatcher(host string, port string, endpoint string, payload []byte) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%s%s", host, port, endpoint), bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	_, err = client.Do(req)
	return err
}
