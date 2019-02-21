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

const (
	LabelStorageGridConfig = "storageGridConfig"
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
func filterEvent(notification *storageGridNotification, sg *storageGrid) bool {
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
func filterName(notification *storageGridNotification, sg *storageGrid) bool {
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

// StartConfig runs a configuration
func (ese *StorageGridEventSourceExecutor) StartEventSource(eventSource *gateways.EventSource, eventStream gateways.Eventing_StartEventSourceServer) error {
	defer gateways.Recover(eventSource.Name)

	ese.Log.Info().Str("event-source-name", eventSource.Name).Msg("operating on event source")
	sg, err := parseEventSource(eventSource.Data)
	if err != nil {
		return err
	}

	return gwcommon.ProcessRoute(&gwcommon.RouteConfig{
		Configs: map[string]interface{}{
			LabelStorageGridConfig: sg,
		},
		Webhook: &gwcommon.Webhook{
			Port:     sg.Port,
			Endpoint: gwcommon.FormatWebhookEndpoint(sg.Endpoint),
		},
		Log:                ese.Log,
		EventSource:        eventSource,
		PostActivate:       gwcommon.DefaultPostActivate,
		PostStop:           gwcommon.DefaultPostStop,
		RouteActiveHandler: RouteActiveHandler,
		StartCh:            make(chan struct{}),
	}, helper, eventStream)
}

// routeActiveHandler handles new route
func RouteActiveHandler(writer http.ResponseWriter, request *http.Request, rc *gwcommon.RouteConfig) {
	logger := rc.Log.With().Str("event-source-name", rc.EventSource.Name).Str("endpoint", rc.Webhook.Endpoint).Str("port", rc.Webhook.Port).Str("method", http.MethodHead).Logger()

	if !helper.ActiveEndpoints[rc.Webhook.Endpoint].Active {
		logger.Warn().Msg("inactive route")
		common.SendErrorResponse(writer, "route is not valid")
		return
	}

	rc.Log.Info().Str("event-source-name", rc.EventSource.Name).Str("method", http.MethodHead).Msg("received a request")
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("failed to parse request body")
		common.SendErrorResponse(writer, "failed to parse request body")
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
		logger.Error().Err(err).Msg("failed to unescape request body url")
		return
	}
	b, err := qson.ToJSON(parsedURL)
	if err != nil {
		logger.Error().Err(err).Msg("failed to convert request body in JSON format")
		return
	}

	var notification *storageGridNotification
	err = json.Unmarshal(b, &notification)
	if err != nil {
		logger.Error().Err(err).Msg("failed to unmarshal request body")
		return
	}

	storageGridConfig := rc.Configs[LabelStorageGridConfig].(*storageGrid)

	if filterEvent(notification, storageGridConfig) && filterName(notification, storageGridConfig) {
		logger.Info().Msg("new event received, dispatching to gateway client")
		helper.ActiveEndpoints[rc.Webhook.Endpoint].DataCh <- b
		return
	}

	logger.Warn().Interface("notification", notification).
		Msg("discarding notification since it did not pass all filters")
}
