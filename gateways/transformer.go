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
	"net"
	"net/http"
	"time"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	pc "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/pkg/errors"
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
func (gatewayCfg *GatewayConfig) DispatchEvent(gatewayEvent *Event) error {
	transformedEvent, err := gatewayCfg.transformEvent(gatewayEvent)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(transformedEvent)
	if err != nil {
		return errors.Errorf("failed to dispatch event to watchers over http. marshalling failed. err: %+v", err)
	}

	switch gatewayCfg.gateway.Spec.EventProtocol.Type {
	case pc.HTTP:
		if err = gatewayCfg.dispatchEventOverHttp(transformedEvent.Context.Source.Host, payload); err != nil {
			return err
		}
	case pc.NATS:
		if err = gatewayCfg.dispatchEventOverNats(transformedEvent.Context.Source.Host, payload); err != nil {
			return err
		}
	default:
		return errors.Errorf("unknown dispatch mechanism %s", gatewayCfg.gateway.Spec.EventProtocol.Type)
	}
	return nil
}

// transformEvent transforms an event from event source into a CloudEvents specification compliant event
// See https://github.com/cloudevents/spec for more info.
func (gatewayCfg *GatewayConfig) transformEvent(gatewayEvent *Event) (*apicommon.Event, error) {
	// Generate an event id
	eventId := suuid.NewV1()

	logger := gatewayCfg.Logger.WithField(common.LabelEventSource, gatewayEvent.Name)

	logger.Infoln("converting gateway event into cloudevents specification compliant event")

	// Create an CloudEvent
	ce := &apicommon.Event{
		Context: apicommon.EventContext{
			CloudEventsVersion: common.CloudEventsVersion,
			EventID:            fmt.Sprintf("%x", eventId),
			ContentType:        "application/json",
			EventTime:          metav1.MicroTime{Time: time.Now().UTC()},
			EventType:          string(gatewayCfg.gateway.Spec.Type),
			EventTypeVersion:   gatewayCfg.gateway.Spec.Version,
			Source: &apicommon.URI{
				Host: common.DefaultEventSourceName(gatewayCfg.gateway.Name, gatewayEvent.Name),
			},
		},
		Payload: gatewayEvent.Payload,
	}

	logger.Infoln("event has been transformed into cloud event")
	return ce, nil
}

// dispatchEventOverHttp dispatches event to watchers over http.
func (gatewayCfg *GatewayConfig) dispatchEventOverHttp(source string, eventPayload []byte) error {
	gatewayCfg.Logger.WithField(common.LabelEventSource, source).Infoln("dispatching event to watchers")

	completeSuccess := true

	for _, sensor := range gatewayCfg.gateway.Spec.Watchers.Sensors {
		namespace := gatewayCfg.Namespace
		if sensor.Namespace != "" {
			namespace = sensor.Namespace
		}
		if err := gatewayCfg.postCloudEventToWatcher(common.ServiceDNSName(sensor.Name, namespace), gatewayCfg.gateway.Spec.EventProtocol.Http.Port, common.SensorServiceEndpoint, eventPayload); err != nil {
			gatewayCfg.Logger.WithField(common.LabelSensorName, sensor.Name).WithError(err).Warnln("failed to dispatch event to sensor watcher over http. communication error")
			completeSuccess = false
		}
	}
	for _, gateway := range gatewayCfg.gateway.Spec.Watchers.Gateways {
		namespace := gatewayCfg.Namespace
		if gateway.Namespace != "" {
			namespace = gateway.Namespace
		}
		if err := gatewayCfg.postCloudEventToWatcher(common.ServiceDNSName(gateway.Name, namespace), gateway.Port, gateway.Endpoint, eventPayload); err != nil {
			gatewayCfg.Logger.WithField(common.LabelGatewayName, gateway.Name).WithError(err).Warnln("failed to dispatch event to gateway watcher over http. communication error")
			completeSuccess = false
		}
	}

	response := "dispatched event to all watchers"
	if !completeSuccess {
		response = fmt.Sprintf("%s.%s", response, " although some of the dispatch operations failed, check logs for more info")
	}

	gatewayCfg.Logger.Infoln(response)
	return nil
}

// dispatchEventOverNats dispatches event over nats
func (gatewayCfg *GatewayConfig) dispatchEventOverNats(source string, eventPayload []byte) error {
	var err error

	switch gatewayCfg.gateway.Spec.EventProtocol.Nats.Type {
	case pc.Standard:
		err = gatewayCfg.natsConn.Publish(source, eventPayload)
	case pc.Streaming:
		err = gatewayCfg.natsStreamingConn.Publish(source, eventPayload)
	}

	if err != nil {
		gatewayCfg.Logger.WithField(common.LabelEventSource, source).WithError(err).Errorln("failed to publish event")
		return err
	}

	gatewayCfg.Logger.WithField(common.LabelEventSource, source).Infoln("event published successfully")
	return nil
}

// postCloudEventToWatcher makes a HTTP POST call to watcher's service
func (gatewayCfg *GatewayConfig) postCloudEventToWatcher(host string, port string, endpoint string, payload []byte) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%s%s", host, port, endpoint), bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				KeepAlive: 600 * time.Second,
			}).Dial,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 50,
		},
	}
	_, err = client.Do(req)
	return err
}
