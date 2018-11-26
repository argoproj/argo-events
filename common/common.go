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
	// EnvVarKubeConfig is the path to the Kubernetes configuration
	EnvVarKubeConfig = "KUBE_CONFIG"

	// SuccessResponse for http request
	SuccessResponse = "Success"
	// ErrorResponse for http request
	ErrorResponse = "Error"

	// StandardTimeFormat is time format reference for golang
	StandardTimeFormat = "2006-01-02 15:04:05"

	// StandardYYYYMMDDFormat formats date in yyyy-mm-dd format
	StandardYYYYMMDDFormat = "2006-01-02"

	// DefaultControllerNamespace is the default namespace where the sensor and gateways controllers are installed
	DefaultControllerNamespace = "argo-events"

	// ImageVersionLatest is the latest tag for image to pull
	ImageVersionLatest = "latest"
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
)

// SENSOR CONSTANTS
const (
	// Sensor image is the image used to deploy sensor.
	SensorImage = "argoproj/sensor"

	// Sensor service port
	SensorServicePort = "9300"

	// the port name on the sensor deployment
	SensorDeploymentPortName = "http"

	// SensorServiceEndpoint is the endpoint to dispatch the event to
	SensorServiceEndpoint = "/"

	// SensorName refers env var for name of sensor
	SensorName = "SENSOR_NAME"

	// SensorNamespace is used to get namespace where sensors are deployed
	SensorNamespace = "SENSOR_NAMESPACE"

	// LabelSensorName is label for sensor name
	LabelSensorName = "sensor-name"

	// LabelSignalName is label for signal name
	LabelSignalName = "signal-name"

	// EnvVarSensorControllerInstanceID is used to get sensor controller instance id
	EnvVarSensorControllerInstanceID = "SENSOR_CONTROLLER_INSTANCE_ID"
)

// GATEWAY CONSTANTS
const (
	// DefaultGatewayControllerDeploymentName is the default deployment name of the gateway-controller-controller
	DefaultGatewayControllerDeploymentName = "gateway-controller"

	// EnvVarGatewayControllerConfigMap contains name of the configmap to retrieve gateway-controller configuration from
	EnvVarGatewayControllerConfigMap = "GATEWAY_CONTROLLER_CONFIG_MAP"

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

	// GatewayNamespace is namespace where gateway controller is deployed
	GatewayNamespace = "GATEWAY_NAMESPACE"

	// LabelGatewayConfigurationName is the label for a configuration in gateway
	LabelGatewayConfigurationName = "config-name"

	// EnvVarGatewayControllerInstanceID is used to get gateway controller instance id
	EnvVarGatewayControllerInstanceID = "GATEWAY_CONTROLLER_INSTANCE_ID"

	// EnvVarGatewayControllerName is used to get name of gateway controller
	EnvVarGatewayControllerName = "GATEWAY_CONTROLLER_NAME"

	// LabelGatewayConfigID is the label for gateway configuration ID
	LabelGatewayConfigID = "GATEWAY_CONFIG_ID"

	// LabelGatewayConfigTimeID is the label for gateway configuration time ID
	LabelGatewayConfigTimeID = "GATEWAY_CONFIG_TIME_ID"
)

// Gateway Processor constants
const (
	// EnvVarGatewayProcessorConfigMap is used to get map containing configurations to run in a gateway
	EnvVarGatewayProcessorConfigMap = "GATEWAY_PROCESSOR_CONFIG_MAP"

	// GatewayProcessorGRPCServerPort is used to get grpc server port for gateway processor server
	GatewayProcessorGRPCServerPort = "GATEWAY_PROCESSOR_GRPC_SERVER_PORT"

	// EnvVarGatewayProcessorClientHTTPPort is used to get http server port for gateway processor client
	EnvVarGatewayProcessorClientHTTPPort = "GATEWAY_PROCESSOR_CLIENT_HTTP_PORT"

	// GatewayProcessorClientHTTPPort is gateway processor client http server port
	GatewayProcessorClientHTTPPort = "9393"

	// EnvVarGatewayProcessorServerHTTPPort is used get http server port for gateway processor server
	EnvVarGatewayProcessorServerHTTPPort = "GATEWAY_PROCESSOR_SERVER_HTTP_PORT"

	// EnvVarGatewayProcessorHTTPServerConfigStartEndpoint is used to get REST endpoint to post new configuration to
	EnvVarGatewayProcessorHTTPServerConfigStartEndpoint = "GATEWAY_HTTP_CONFIG_START"

	// GatewayProcessorHTTPServerConfigStartEndpoint is REST endpoint to post new configuration to
	GatewayProcessorHTTPServerConfigStartEndpoint = "/start"

	// EnvVarGatewayProcessorHTTPServerConfigStopEndpoint is used to get REST endpoint to post to stop a configuration
	EnvVarGatewayProcessorHTTPServerConfigStopEndpoint = "GATEWAY_HTTP_CONFIG_STOP"

	// GatewayProcessorHTTPServerConfigStopEndpoint is REST endpoint to post to stop a configuration
	GatewayProcessorHTTPServerConfigStopEndpoint = "/stop"

	// EnvVarGatewayProcessorHTTPServerEventEndpoint is used to get the REST endpoint to send event to for gateway processor server
	EnvVarGatewayProcessorHTTPServerEventEndpoint = "GATEWAY_HTTP_CONFIG_EVENT"

	// GatewayProcessorHTTPServerEventEndpoint is REST endpoint to send event to for gateway processor server
	GatewayProcessorHTTPServerEventEndpoint = "/event"

	// EnvVarGatewayProcessorHTTPServerConfigActivated is used to get the REST endpoint to send  notifications for configuration that are successfully running
	EnvVarGatewayProcessorHTTPServerConfigActivated = "GATEWAY_HTTP_CONFIG_RUNNING"

	// GatewayProcessorHTTPServerConfigActivatedEndpoint is the REST endpoint on which gateway processor listens for activated notifications from configurations
	GatewayProcessorHTTPServerConfigActivatedEndpoint = "/activated"

	// EnvVarGatewayProcessorHTTPServerConfigError is used to get the REST endpoint on which gateway processor listens for errors from configurations
	EnvVarGatewayProcessorHTTPServerConfigError = "GATEWAY_HTTP_CONFIG_ERROR"

	// GatewayProcessorHTTPServerConfigErrorEndpoint is the REST endpoint on which gateway processor listens for errors from configurations
	GatewayProcessorHTTPServerConfigErrorEndpoint = "/error"

	// GatewayProcessorGRPCClientImage is gRPC gateway processor client image
	GatewayProcessorGRPCClientImage = "argoproj/gateway-processor-grpc-client"

	// GatewayProcessorHTTPClientImage is HTTP gateway processor client image
	GatewayProcessorHTTPClientImage = "argoproj/gateway-processor-http-client"
)

// Gateway Transformer constants
const (
	// EnvVarGatewayTransformerConfigMap is used for gateway  configuration
	EnvVarGatewayTransformerConfigMap = "GATEWAY_TRANSFORMER_CONFIG_MAP"

	// GatewayHTTPEventTransformerImage is image for gateway http event transformer
	GatewayHTTPEventTransformerImage = "argoproj/gateway-http-transformer"

	// GatewayNATSEventTransformerImage is image for gateway nats event transformer
	GatewayNATSEventTransformerImage = "argoproj/gateway-nats-transformer"

	// GatewayKafkaEventTransformerImage is image for gateway kafka event transformer
	GatewayKafkaEventTransformerImage = "argoproj/gateway-kafka-transformer"

	//  EnvVarGatewayTransformerPort is the env var for http server port
	EnvVarGatewayTransformerPort = "TRANSFORMER_PORT"

	// TransformerPort is http server port where transformer service is running
	GatewayTransformerPort = "9300"

	// EventTypeEnvVar contains the type of event
	EventType = "EVENT_TYPE"

	// EnvVarEventTypeVersion contains the event type version
	EventTypeVersion = "EVENT_TYPE_VERSION"

	// EnvVarEventSource contains the name of the gateway
	EventSource = "EVENT_SOURCE"

	// SensorWatchers is the list of sensors interested in listening to gateway notifications
	SensorWatchers = "SENSOR_WATCHERS"

	// GatewayWatchers is the list of gateways interested in listening to gateway notifications
	GatewayWatchers = "GATEWAY_WATCHERS"
)

// CloudEvents constants
const (
	// CloudEventsVersion is the version of the CloudEvents spec targeted+
	// by this library.
	CloudEventsVersion = "0.1"
)
