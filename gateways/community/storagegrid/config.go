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
	"net/http"
	"time"

	"github.com/argoproj/argo-events/gateways/common"

	"github.com/ghodss/yaml"
	"github.com/rs/zerolog"
)

// StorageGridEventSourceExecutor implements Eventing
type StorageGridEventSourceExecutor struct {
	Log zerolog.Logger
}

// storageGrid contains configuration for storage grid sns
type storageGrid struct {
	// Webhook
	Hook *common.Webhook `json:"hook"`

	// Events are s3 bucket notification events.
	// For more information on s3 notifications, follow https://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html#notification-how-to-event-types-and-destinations
	// Note that storage grid notifications do not contain `s3:`
	Events []string `json:"events,omitempty"`

	// Filter on object key which caused the notification.
	Filter *Filter `json:"filter,omitempty"`

	// srv holds reference to http server
	srv *http.Server
	mux *http.ServeMux
}

// Filter represents filters to apply to bucket notifications for specifying constraints on objects
// +k8s:openapi-gen=true
type Filter struct {
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}

// storageGridNotification is the bucket notification received from storage grid
type storageGridNotification struct {
	Action  string `json:"Action"`
	Message struct {
		Records []struct {
			EventVersion string    `json:"eventVersion"`
			EventSource  string    `json:"eventSource"`
			EventTime    time.Time `json:"eventTime"`
			EventName    string    `json:"eventName"`
			UserIdentity struct {
				PrincipalID string `json:"principalId"`
			} `json:"userIdentity"`
			RequestParameters struct {
				SourceIPAddress string `json:"sourceIPAddress"`
			} `json:"requestParameters"`
			ResponseElements struct {
				XAmzRequestID string `json:"x-amz-request-id"`
			} `json:"responseElements"`
			S3 struct {
				S3SchemaVersion string `json:"s3SchemaVersion"`
				ConfigurationID string `json:"configurationId"`
				Bucket          struct {
					Name          string `json:"name"`
					OwnerIdentity struct {
						PrincipalID string `json:"principalId"`
					} `json:"ownerIdentity"`
					Arn string `json:"arn"`
				} `json:"bucket"`
				Object struct {
					Key       string `json:"key"`
					Size      int    `json:"size"`
					ETag      string `json:"eTag"`
					Sequencer string `json:"sequencer"`
				} `json:"object"`
			} `json:"s3"`
		} `json:"Records"`
	} `json:"Message"`
	TopicArn string `json:"TopicArn"`
	Version  string `json:"Version"`
}

func parseEventSource(eventSource string) (interface{}, error) {
	var s *storageGrid
	err := yaml.Unmarshal([]byte(eventSource), &s)
	if err != nil {
		return nil, err
	}
	return s, err
}
