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

package aws_sns

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/ghodss/yaml"
	"net/http"
)

// AWSSNSConfig contains information to configure sns notifications
type AWSSNSConfig struct {
	// Port is the http server port to which the endpoint should be associated with
	Port string `json:"port"`

	// Endpoint to listen on for SNS notification
	Endpoint string `json:"endpoint"`

	// CompletePayload if set true makes gateway dispatch complete sns notification else
	// only the message.
	// For more information on sns notification, refer https://docs.aws.amazon.com/sns/latest/dg/sns-http-https-endpoint-as-subscriber.html#SendMessageToHttp.prepare
	CompletePayload bool `json:"completePayload"`

	// +k8s:openapi-gen=false
	srv *http.Server
	// +k8s:openapi-gen=false
	mux *http.ServeMux
}

// AWSSNSConfigExecutor implements ConfigExecutor
type AWSSNSConfigExecutor struct {
	*gateways.GatewayConfig
	snsClient *sns.SNS
}

// parseConfig parses a configuration of gateway
func parseConfig(config string) (*AWSSNSConfig, error) {
	var a *AWSSNSConfig
	err := yaml.Unmarshal([]byte(config), &a)
	if err != nil {
		return nil, err
	}
	return a, err
}
