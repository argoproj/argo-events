/*
Copyright 2020 BlackRock, Inc.

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
	"reflect"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

// Defaults
const (
	// DefaultControllerNamespace is the default namespace where the sensor and gateways controllers are installed
	DefaultControllerNamespace = "argo-events"
)

// Environment variables
const (
	// EnvVarKubeConfig is the path to the Kubernetes configuration
	EnvVarKubeConfig = "KUBE_CONFIG"
	// EnvVarDebugLog is the env var to turn on the debug mode for logging
	EnvVarDebugLog = "DEBUG_LOG"
)

// Controller environment variables
const (
	// EnvVarControllerConfigMap contains name of the configmap to retrieve controller configuration from
	EnvVarControllerConfigMap = "CONTROLLER_CONFIG_MAP"
	// EnvVarControllerInstanceID is used to get controller instance id
	EnvVarControllerInstanceID = "CONTROLLER_INSTANCE_ID"
	// EnvVarNamespace refers to a K8s namespace
	EnvVarNamespace = "NAMESPACE"
	// EnvVarGatewayClientImage refers to the env var for gateway client image
	EnvVarGatewayClientImage = "GATEWAY_CLIENT_IMAGE"
	// EnvVarGatewayServerImage refers to the default gateway server image
	EnvVarGatewayServerImage = "GATEWAY_SERVER_IMAGE"
	// EnvVarSensorObject refers to the env of based64 encoded sensor spec
	EnvVarSensorObject = "SENSOR_OBJECT"
	// EnvVarEventSourceObject refers to the env of based64 encoded eventsource spec
	EnvVarEventSourceObject = "EVENTSOURCE_OBJECT"
)

// EventBus related
const (
	// EnvVarEventBusConfig refers to the eventbus config env
	EnvVarEventBusConfig = "EVENTBUS_CONFIG"
	// EnvVarEventBusSubject refers to the eventbus subject env
	EnvVarEventBusSubject = "EVENTBUS_SUBJECT"
	// volumeMount path for eventbus auth file
	EventBusAuthFileMountPath = "/etc/eventbus/auth"
)

// Controller labels
const (
	// LabelResourceName is the label for the K8s resource name
	LabelResourceName = "resource-name"
)

const (
	// GatewayControllerConfigMapKey is the key in the configmap to retrieve controller configuration from.
	// Content encoding is expected to be YAML.
	ControllerConfigMapKey = "config"
)

// Sensor constants
const (
	// SensorNamespace is used to get namespace where sensors are deployed
	SensorNamespace = "SENSOR_NAMESPACE"
	// LabelSensorName is label for sensor name
	LabelSensorName = "sensor-name"
)

// Gateway constants
const (
	// LabelEventSourceName is the label for a event source
	LabelEventSourceName = "eventsource-name"
)

const (
	// EnvVarEventSource refers to event source name
	EnvVarEventSource = "EVENT_SOURCE"
	// AnnotationResourceSpecHash is the annotation of a K8s resource spec hash
	AnnotationResourceSpecHash = "resource-spec-hash"
)

var (
	ErrNilEventSource = errors.New("event source can't be nil")
)

// Miscellaneous Labels
const (
	// LabelEventSource is label for event name
	LabelEventSource = "event-source"
	// LabelOwnerName is the label for resource owner name
	LabelOwnerName = "owner-name"
)

// various supported media types
const (
	MediaTypeJSON string = "application/json"
	MediaTypeYAML string = "application/yaml"
)

var (
	SecretKeySelectorType    = reflect.TypeOf(&corev1.SecretKeySelector{})
	ConfigMapKeySelectorType = reflect.TypeOf(&corev1.ConfigMapKeySelector{})
)
