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

package common

import (
	"github.com/argoproj/argo-events/pkg/apis/gateway"
	"github.com/argoproj/argo-events/pkg/apis/sensor"
)

const (
	// EnvVarNamespace contains the namespace of the controller & services
	EnvVarNamespace = "ARGO_EVENTS_NAMESPACE"

	// EnvVarKubeConfig is the path to the Kubernetes configuration
	EnvVarKubeConfig = "KUBE_CONFIG"

	// http responses
	SuccessResponse = "Success"
	ErrorResponse   = "Error"
)

// SENSOR CONTROLLER CONSTANTS
const (
	// DefaultSensorControllerDeploymentName is the default deployment name of the sensor sensor-controller
	DefaultSensorControllerDeploymentName = "sensor-controller"

	// SensorControllerConfigMapKey is the key in the configmap to retrieve sensor configuration from.
	// Content encoding is expected to be YAML.
	SensorControllerConfigMapKey = "config"

	//LabelKeySensorControllerInstanceID is the label which allows to separate application among multiple running sensor controllers.
	LabelKeySensorControllerInstanceID = sensor.FullName + "/sensor-controller-instanceid"

	// LabelKeyPhase is a label applied to sensors to indicate the current phase of the sensor (for filtering purposes)
	LabelKeyPhase = sensor.FullName + "/phase"

	// LabelKeyComplete is the label to mark sensors as complete
	LabelKeyComplete = sensor.FullName + "/complete"

	// EnvVarConfigMap is the name of the configmap to use for the sensor-controller
	EnvVarConfigMap = "SENSOR_CONFIG_MAP"

	// Sensor image is the image used to deploy sensor.
	SensorImage = "metalgearsolid/sensor"

	// Sensor service port
	SensorServicePort = 9300

	// SensorName refers env var for name of sensor
	SensorName = "SENSOR_NAME"

	SensorNamespace = "SENSOR_NAMESPACE"

	// LabelJobName is label for job name
	LabelJobName = "job-name"
)

// GATEWAY CONSTANTS
const (
	// DefaultGatewayControllerDeploymentName is the default deployment name of the gateway-controller-controller
	DefaultGatewayControllerDeploymentName = "gateway-controller"

	// GatewayControllerConfigMapEnvVar contains name of the configmap to retrieve gateway-controller configuration from
	GatewayControllerConfigMapEnvVar = "GATEWAY_CONTROLLER_CONFIG_MAP"

	// GatewayControllerConfigMapKey is the key in the configmap to retrieve gateway-controller configuration from.
	// Content encoding is expected to be YAML.
	GatewayControllerConfigMapKey = "config"

	//LabelKeyGatewayControllerInstanceID is the label which allows to separate application among multiple running gateway-controller controllers.
	LabelKeyGatewayControllerInstanceID = gateway.FullName + "/gateway-controller-instanceid"

	// GatewayLabelKeyPhase is a label applied to gateways to indicate the current phase of the gateway-controller (for filtering purposes)
	GatewayLabelKeyPhase = gateway.FullName + "/phase"

	// LabelGatewayName is the label for gateway name
	LabelGatewayName = "gateway-name"

	// GatewayName refers env var for name of gateway
	GatewayName = "GATEWAY_NAME"
)

// Gateway Processor constants
const (
	GatewayProcessorConfigMapEnvVar = "GATEWAY_PROCESSOR_CONFIG_MAP"

	GatewayProcessorGRPCServerPort = "GATEWAY_PROCESSOR_GRPC_SERVER_PORT"

	GatewayProcessorClientHTTPPortEnvVar = "GATEWAY_PROCESSOR_CLIENT_HTTP_PORT"

	GatewayProcessorClientHTTPPort = "9393"

	GatewayProcessorServerHTTPPortEnvVar = "GATEWAY_PROCESSOR_SERVER_HTTP_PORT"

	GatewayProcessorHTTPServerConfigStartEndpointEnvVar = "GATEWAY_HTTP_CONFIG_START"

	GatewayProcessorHTTPServerConfigStartEndpoint = "/start"

	GatewayProcessorHTTPServerConfigStopEndpointEnvVar = "GATEWAY_HTTP_CONFIG_STOP"

	GatewayProcessorHTTPServerConfigStopEndpoint = "/stop"

	GatewayProcessorHTTPServerEventEndpointEnvVar = "GATEWAY_HTTP_CONFIG_EVENT"

	GatewayProcessorHTTPServerEventEndpoint = "/event"

	GatewayProcessorGRPCClientImage = "metalgearsolid/gateway-processor-grpc-client"

	GatewayProcessorHTTPClientImage = "metalgearsolid/gateway-processor-http-client"
)

// Gateway Transformer constants
const (
	// GatewayConfigMapEnvVar is used for gateway  configuration
	GatewayTransformerConfigMapEnvVar = "GATEWAY_TRANSFORMER_CONFIG_MAP"

	// GatewayEventTransformerImage is image for gateway event transformer
	GatewayEventTransformerImage = "metalgearsolid/gateway-transformer"

	//  TransformerPortEnvVar is the env var for http server port
	GatewayTransformerPortEnvVar = "TRANSFORMER_PORT"

	// TransformerPort is http server port where transformer service is running
	GatewayTransformerPort = 9300

	// EventTypeEnvVar contains the type of event
	EventType = "EVENT_TYPE"

	// EnvVarEventTypeVersion contains the event type version
	EventTypeVersion = "EVENT_TYPE_VERSION"

	// EnvVarEventSource contains the name of the gateway
	EventSource = "EVENT_SOURCE"

	// SensorList is list of sensor to dispatch message to
	SensorList = "SENSOR_LIST"
)

// CloudEvents constants
const (
	// CloudEventsVersion is the version of the CloudEvents spec targeted
	// by this library.
	CloudEventsVersion = "0.1"

	// HeaderContentType is the standard HTTP header "Content-Type"
	HeaderContentType = "Content-Type"
)
