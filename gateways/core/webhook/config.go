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

package webhook

import (
	"net/http"

	"github.com/ghodss/yaml"
	"github.com/rs/zerolog"
)

// WebhookEventSourceExecutor implements Eventing
type WebhookEventSourceExecutor struct {
	Log zerolog.Logger
}

// webhook is a general purpose REST API
// +k8s:openapi-gen=true
type webhook struct {
	// REST API endpoint
	Endpoint string `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`
	// Method is HTTP request method that indicates the desired action to be performed for a given resource.
	// See RFC7231 Hypertext Transfer Protocol (HTTP/1.1): Semantics and Content
	Method string `json:"method" protobuf:"bytes,2,opt,name=method"`
	// Port on which HTTP server is listening for incoming events.
	Port string `json:"port" protobuf:"bytes,3,opt,name=port"`
	// srv holds reference to http server
	// +k8s:openapi-gen=false
	srv *http.Server `json:"srv,omitempty"`
	// +k8s:openapi-gen=false
	mux *http.ServeMux `json:"mux,omitempty"`
}

func parseEventSource(es string) (*webhook, error) {
	var n *webhook
	err := yaml.Unmarshal([]byte(es), &n)
	if err != nil {
		return nil, err
	}
	return n, nil
}
