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

import "net/http"

// StorageGridEventConfig contains configuration for storage grid sns
// +k8s:openapi-gen=true
type StorageGridEventConfig struct {
	// Port to run web server on
	Port     string `json:"port"`
	// Endpoint to listen to events on
	Endpoint string `json:"endpoint"`
	// Events are s3 bucket notification events.
	// For more information on s3 notifications, follow https://docs.aws.amazon.com/AmazonS3/latest/dev/NotificationHowTo.html#notification-how-to-event-types-and-destinations
	// Note that storage grid notifications do not contain `s3:`
	Events []string `json:"events,omitempty"`
	// Filter on object key which caused the notification.
	Filter *Filter `json:"filter,omitempty"`
	// srv holds reference to http server
	// +k8s:openapi-gen=false
	Srv *http.Server
	// +k8s:openapi-gen=false
	Mux *http.ServeMux
}

// Filter represents filters to apply to bucket nofifications for specifying constraints on objects
// +k8s:openapi-gen=true
type Filter struct {
	Prefix string `json:"prefix"`
	Suffix string `json:"suffix"`
}
