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
	"github.com/joncalhoun/qson"
	"github.com/satori/go.uuid"
)

var (
	helper = gwcommon.NewWebhookHelper()

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
	go gwcommon.InitRouteChannels(helper)
}

// generateUUID returns a new uuid
func generateUUID() uuid.UUID {
	return uuid.NewV4()
}

// filterEvent filters notification based on event filter in a gateway configuration
func filterEvent(notification *storageGridNotification, sg *storageGridEventSource) bool {
	if sg.Events == nil {
		return true
	}
	for _, filterEvent := range sg.Events {
		if notification.Message.Records[0].EventName == filterEvent {
			return true
		}
	}
	return false
}

// filterName filters object key based on configured prefix and/or suffix
func filterName(notification *storageGridNotification, sg *storageGridEventSource) bool {
	if sg.Filter == nil {
		return true
	}
	if sg.Filter.Prefix != "" && sg.Filter.Suffix != "" {
		return strings.HasPrefix(notification.Message.Records[0].S3.Object.Key, sg.Filter.Prefix) && strings.HasSuffix(notification.Message.Records[0].S3.Object.Key, sg.Filter.Suffix)
	}
	if sg.Filter.Prefix != "" {
		return strings.HasPrefix(notification.Message.Records[0].S3.Object.Key, sg.Filter.Prefix)
	}
	if sg.Filter.Suffix != "" {
		return strings.HasSuffix(notification.Message.Records[0].S3.Object.Key, sg.Filter.Suffix)
	}
	return true
}

func (rc *RouteConfig) GetRoute() *gwcommon.Route {
	return rc.route
}

// StartConfig runs a configuration
func (ese *StorageGridEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	log := ese.Log.WithEventSource(eventSource.Name)

	log.Info("operating on event source")
	config, err := parseEventSource(eventSource.Data)
	if err != nil {
		log.WithError(err).Error("failed to parse event source")
		return err
	}
	sges := config.(*storageGridEventSource)

	return gwcommon.ProcessRoute(&RouteConfig{
		route: &gwcommon.Route{
			Webhook:     sges.Hook,
			EventSource: eventSource,
			Logger:      ese.Log,
			StartCh:     make(chan struct{}),
		},
		sges: sges,
	}, helper, eventStream)
}

func (rc *RouteConfig) PostStart() error {
	return nil
}

func (rc *RouteConfig) PostStop() error {
	return nil
}

// RouteHandler handles new route
func (rc *RouteConfig) RouteHandler(writer http.ResponseWriter, request *http.Request) {
	log := rc.route.Logger.
		WithEventSource(rc.route.EventSource.Name).
		WithEndpoint(rc.route.Webhook.Endpoint).
		WithPort(rc.route.Webhook.Port).
		WithHttpMethod(request.Method)

	if !helper.ActiveEndpoints[rc.route.Webhook.Endpoint].Active {
		log.Warn("inactive route")
		common.SendErrorResponse(writer, "")
		return
	}

	log.Info("received a request")
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.WithError(err).Error("failed to parse request body")
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
		log.WithError(err).Error("failed to unescape request body url")
		return
	}
	b, err := qson.ToJSON(parsedURL)
	if err != nil {
		log.WithError(err).Error("failed to convert request body in JSON format")
		return
	}

	var notification *storageGridNotification
	err = json.Unmarshal(b, &notification)
	if err != nil {
		log.WithError(err).Error("failed to unmarshal request body")
		return
	}

	if filterEvent(notification, rc.sges) && filterName(notification, rc.sges) {
		log.WithError(err).Error("new event received, dispatching to gateway client")
		helper.ActiveEndpoints[rc.route.Webhook.Endpoint].DataCh <- b
		return
	}

	log.Warn("discarding notification since it did not pass all filters")
}
