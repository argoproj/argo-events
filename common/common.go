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

// Environment variables
const (
	// EnvVarKubeConfig is the path to the Kubernetes configuration
	EnvVarKubeConfig = "KUBE_CONFIG"
	// EnvVarDebugLog is the env var to turn on the debug mode for logging
	EnvVarDebugLog = "DEBUG_LOG"
)

// EventBus related
const (
	// EnvVarEventBusConfig refers to the eventbus config env
	EnvVarEventBusConfig = "EVENTBUS_CONFIG"
	// EnvVarEventBusSubject refers to the eventbus subject env
	EnvVarEventBusSubject = "EVENTBUS_SUBJECT"
	// volumeMount path for eventbus auth file
	EventBusAuthFileMountPath = "/etc/eventbus/auth"
	// Default NATS Streaming messages max age
	NATSStreamingMaxAge = "72h"
	// Default EventBus name
	DefaultEventBusName = "default"
)

// Sensor constants
const (
	// EnvVarSensorObject refers to the env of based64 encoded sensor spec
	EnvVarSensorObject = "SENSOR_OBJECT"
	// SensorNamespace is used to get namespace where sensors are deployed
	SensorNamespace = "SENSOR_NAMESPACE"
	// LabelSensorName is label for sensor name
	LabelSensorName = "sensor-name"
)

// EventSource
const (
	// EnvVarEventSourceObject refers to the env of based64 encoded eventsource spec
	EnvVarEventSourceObject = "EVENTSOURCE_OBJECT"
	// EnvVarEventSource refers to event source name
	EnvVarEventSource = "EVENT_SOURCE"
	// LabelEventSourceName is the label for a event source
	LabelEventSourceName = "eventsource-name"
)

var (
	ErrNilEventSource = errors.New("event source can't be nil")
)

// Miscellaneous Labels
const (
	// LabelOwnerName is the label for resource owner name
	LabelOwnerName = "owner-name"
	// AnnotationResourceSpecHash is the annotation of a K8s resource spec hash
	AnnotationResourceSpecHash = "resource-spec-hash"
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
