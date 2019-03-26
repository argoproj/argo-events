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

package sensors

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor"
)

// HttpEventProtocol handles events sent over HTTP
func (sec *sensorExecutionCtx) HttpEventProtocol() {
	// create a http server. this server listens for events from gateway.
	sec.server = &http.Server{Addr: fmt.Sprintf(":%s", sec.sensor.Spec.EventProtocol.Http.Port)}

	// add a handler to handle incoming events
	http.HandleFunc("/", sec.httpEventHandler)

	sec.log.WithPort(sec.sensor.Spec.EventProtocol.Http.Port).Info("sensor started listening")
	if err := sec.server.ListenAndServe(); err != nil {
		sec.log.WithError(err).Error("sensor server stopped")
		// escalate error
		labels := map[string]string{
			common.LabelEventType:  string(common.EscalationEventType),
			common.LabelSensorName: sec.sensor.Name,
			common.LabelOperation:  "server_shutdown",
		}
		if err := common.GenerateK8sEvent(sec.kubeClient, fmt.Sprintf("sensor server stopped"), common.EscalationEventType,
			"server shutdown", sec.sensor.Name, sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
			sec.log.WithError(err).Error("failed to create K8s event to log server shutdown error")
		}
	}
}

// Handles events received from gateways sent over http
func (sec *sensorExecutionCtx) httpEventHandler(w http.ResponseWriter, r *http.Request) {
	var response string

	sec.log.Info("received an event from gateway")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		response = "failed to read request body"
		sec.log.WithError(err).Error(response)
		common.SendErrorResponse(w, response)
	}

	event, err := sec.parseEvent(body)
	if err != nil {
		response = "failed to parse request into event"
		sec.log.WithError(err).Error(response)
		common.SendErrorResponse(w, response)
	}

	// validate whether the event is from gateway that this sensor is watching and send event over internal queue if valid
	if sec.sendEventToInternalQueue(event, w) {
		response = "message successfully sent over internal queue"
		sec.log.WithEventSource(event.Context.Source.Host).Info(response)
		common.SendSuccessResponse(w, response)
		return
	}

	response = "event is from unknown source"
	sec.log.WithEventSource(event.Context.Source.Host).Warn(response)
	common.SendErrorResponse(w, response)
}
