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
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	pc "github.com/argoproj/argo-events/pkg/apis/common"
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

// DispatchEvent dispatches event to gateway transformer for further processing
func (gc *GatewayConfig) DispatchEvent(gatewayEvent *Event) error {
	transformedEvent, err := gc.transformEvent(gatewayEvent)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(transformedEvent)
	if err != nil {
		return fmt.Errorf("failed to dispatch event to watchers over http. marshalling failed. err: %+v", err)
	}

	switch gc.gw.Spec.EventProtocol.Type {
	case pc.HTTP:
		if err = gc.dispatchEventOverHttp(transformedEvent.Context.Source.Host, payload); err != nil {
			return err
		}
	case pc.NATS:
		if err = gc.dispatchEventOverNats(transformedEvent.Context.Source.Host, payload); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown dispatch mechanism %s", gc.gw.Spec.EventProtocol)
	}
	return nil
}

// transformEvent transforms an event from event source into a CloudEvents specification compliant event
// See https://github.com/cloudevents/spec for more info.
func (gc *GatewayConfig) transformEvent(gatewayEvent *Event) (*apicommon.Event, error) {
	// Generate an event id
	eventId := suuid.NewV1()

	gc.Log.Info().Str("source", gatewayEvent.Name).
		Msg("converting gateway event into cloudevents specification compliant event")

	// Create an CloudEvent
	ce := &apicommon.Event{
		Context: apicommon.EventContext{
			CloudEventsVersion: common.CloudEventsVersion,
			EventID:            fmt.Sprintf("%x", eventId),
			ContentType:        "application/json",
			EventTime:          metav1.MicroTime{Time: time.Now().UTC()},
			EventType:          gc.gw.Spec.Type,
			EventTypeVersion:   gc.gw.Spec.EventVersion,
			Source: &apicommon.URI{
				Host: common.DefaultEventSourceName(gc.gw.Name, gatewayEvent.Name),
			},
		},
		Payload: gatewayEvent.Payload,
	}

	gc.Log.Info().Str("event-source", gatewayEvent.Name).Msg("event has been transformed into cloud event")
	return ce, nil
}

// dispatchEventOverHttp dispatches event to watchers over http.
func (gc *GatewayConfig) dispatchEventOverHttp(source string, eventPayload []byte) error {
	gc.Log.Info().Str("source", source).Msg("dispatching event to watchers")

	completeSuccess := true

	for _, sensor := range gc.gw.Spec.Watchers.Sensors {
		if err := gc.postCloudEventToWatcher(common.DefaultServiceName(sensor.Name), gc.gw.Spec.EventProtocol.Http.Port, common.SensorServiceEndpoint, eventPayload); err != nil {
			gc.Log.Warn().Str("event-source", source).Str("sensor-name", sensor.Name).Err(err).Msg("failed to dispatch event to sensor watcher over http. communication error")
			completeSuccess = false
		}
	}
	for _, gateway := range gc.gw.Spec.Watchers.Gateways {
		if err := gc.postCloudEventToWatcher(common.DefaultServiceName(gateway.Name), gateway.Port, gateway.Endpoint, eventPayload); err != nil {
			gc.Log.Warn().Str("event-source", source).Str("gateway-name", gateway.Name).Err(err).Msg("failed to dispatch event to gateway watcher over http. communication error")
			completeSuccess = false
		}
	}

	response := "dispatched event to all watchers"
	if !completeSuccess {
		response = fmt.Sprintf("%s.%s", response, " although some of the dispatch operations failed, check logs for more info")
	}

	gc.Log.Info().Msg(response)
	return nil
}

// dispatchEventOverNats dispatches event over nats
func (gc *GatewayConfig) dispatchEventOverNats(source string, eventPayload []byte) error {
	var err error

	switch gc.gw.Spec.EventProtocol.Nats.Type {
	case pc.Standard:
		err = gc.natsConn.Publish(source, eventPayload)
	case pc.Streaming:
		err = gc.natsStreamingConn.Publish(source, eventPayload)
	}

	if err != nil {
		gc.Log.Error().Err(err).Str("source", source).Msg("failed to publish event")
		return err
	}
	gc.Log.Info().Str("source", source).Msg("event published successfully")
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
