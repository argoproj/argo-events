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

package storagegrid

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	gwcommon "github.com/argoproj/argo-events/gateways/common"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/joncalhoun/qson"
)

var (
	helper = gwcommon.NewWebhookController()

	respBody = `
<PublishResponse xmlns="http://argoevents-sns-server/">
    <PublishResult> 
        <MessageId>` + generateUUID().String() + `</MessageId> 
    </PublishResult> 
    <ResponseMetadata>
       <RequestId>` + generateUUID().String() + `</RequestId>
    </ResponseMetadata> 
</PublishResponse>` + "\n"
)

func init() {
	go gwcommon.InitializeRouteChannels(helper)
}

// generateUUID returns a new uuid
func generateUUID() uuid.UUID {
	return uuid.New()
}

// filterEvent filters notification based on event filter in a gateway configuration
func filterEvent(notification *storageGridNotification, eventSource *v1alpha1.StorageGridEventSource) bool {
	if eventSource.Events == nil {
		return true
	}
	for _, filterEvent := range eventSource.Events {
		if notification.Message.Records[0].EventName == filterEvent {
			return true
		}
	}
	return false
}

// filterName filters object key based on configured prefix and/or suffix
func filterName(notification *storageGridNotification, eventSource *v1alpha1.StorageGridEventSource) bool {
	if eventSource.Filter == nil {
		return true
	}
	if eventSource.Filter.Prefix != "" && eventSource.Filter.Suffix != "" {
		return strings.HasPrefix(notification.Message.Records[0].S3.Object.Key, eventSource.Filter.Prefix) && strings.HasSuffix(notification.Message.Records[0].S3.Object.Key, eventSource.Filter.Suffix)
	}
	if eventSource.Filter.Prefix != "" {
		return strings.HasPrefix(notification.Message.Records[0].S3.Object.Key, eventSource.Filter.Prefix)
	}
	if eventSource.Filter.Suffix != "" {
		return strings.HasSuffix(notification.Message.Records[0].S3.Object.Key, eventSource.Filter.Suffix)
	}
	return true
}

func (rc *Router) GetRoute() *gwcommon.Route {
	return rc.route
}

// StartConfig runs a configuration
func (listener *EventListener) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	log := listener.Logger.WithField(common.LabelEventSource, eventSource.Name)

	log.Info("started processing the event source...")

	var storagegridEventSource *v1alpha1.StorageGridEventSource
	if err := yaml.Unmarshal(eventSource.Value, &storagegridEventSource); err != nil {
		log.WithError(err).Errorln("failed to parse the event source")
		return err
	}

	return gwcommon.ProcessRoute(&Router{
		route: &gwcommon.Route{
			Webhook:     storagegridEventSource.WebHook,
			EventSource: eventSource,
			Logger:      listener.Logger,
			StartCh:     make(chan struct{}),
		},
		eventSource: storagegridEventSource,
	}, helper, eventStream)
}

func (rc *Router) PostStart() error {
	return nil
}

func (rc *Router) PostStop() error {
	return nil
}

// HandleRoute handles new route
func (rc *Router) HandleRoute(writer http.ResponseWriter, request *http.Request) {
	r := rc.route

	log := r.Logger.WithFields(
		map[string]interface{}{
			common.LabelEventSource: r.EventSource.Name,
			common.LabelEndpoint:    r.Webhook.Endpoint,
			common.LabelPort:        r.Webhook.Port,
			common.LabelHTTPMethod:  r.Webhook.Method,
		})

	if !helper.ActiveEndpoints[r.Webhook.Endpoint].Active {
		log.Warn("inactive route")
		common.SendErrorResponse(writer, "")
		return
	}

	log.Info("received a request")
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.WithError(err).Errorln("failed to parse request body")
		common.SendErrorResponse(writer, "")
		return
	}

	switch request.Method {
	case http.MethodHead:
		respBody = ""
	}
	writer.WriteHeader(http.StatusOK)
	writer.Header().Add("Content-Type", "text/plain")
	writer.Write([]byte(respBody))

	// notification received from storage grid is url encoded.
	parsedURL, err := url.QueryUnescape(string(body))
	if err != nil {
		log.WithError(err).Errorln("failed to unescape request body url")
		return
	}
	b, err := qson.ToJSON(parsedURL)
	if err != nil {
		log.WithError(err).Errorln("failed to convert request body in JSON format")
		return
	}

	var notification *storageGridNotification
	err = json.Unmarshal(b, &notification)
	if err != nil {
		log.WithError(err).Errorln("failed to unmarshal request body")
		return
	}

	if filterEvent(notification, rc.eventSource) && filterName(notification, rc.eventSource) {
		log.WithError(err).Errorln("new event received, dispatching to gateway client")
		helper.ActiveEndpoints[rc.route.Webhook.Endpoint].DataCh <- b
		return
	}

	log.Warnln("discarding notification since it did not pass all filters")
}
