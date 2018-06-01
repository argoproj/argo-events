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
	"github.com/blackrock/axis/pkg/apis/sensor"
)

// SENSOR CONTROLLER CONSTANTS
const (
	// DefaultSensorControllerNamespace is the default namespace where the sensor controller is installed
	DefaultSensorControllerNamespace = "default"

	// DefaultSensorControllerDeploymentName is the default deployment name of the sensor controller
	DefaultSensorControllerDeploymentName = "sensor-controller"

	// SensorControllerConfigMapKey is the key in the configmap to retrieve sensor configuration from.
	// Content encoding is expected to be YAML.
	SensorControllerConfigMapKey = "config"

	//LabelKeySensorControllerInstanceID is the label which allows to separate application among multiple running sensor controllers.
	LabelKeySensorControllerInstanceID = sensor.FullName + "/controller-instanceid"

	// LabelKeyPhase is a label applied to sensors to indicate the current phase of the sensor (for filtering purposes)
	LabelKeyPhase = sensor.FullName + "/phase"

	// ExecutorContainerName is the name of the sensor executor container
	ExecutorContainerName = "sensor-executor"

	// LabelKeyResolved is the label to mark sensor jobs as having resolved all dependencies
	LabelKeyResolved = sensor.FullName + "/resolved"

	// LabelJobName is the label to mark sensor job pods in order to find them from the job
	LabelJobName = "job-name"

	// AnnotationKeyNodeMessage is the job metadata annotation key the job executor will use to communicate errors during execution
	AnnotationKeyNodeMessage = sensor.FullName + "/node-message"

	// EnvVarJobName is the name of the job
	EnvVarJobName = "SENSOR_JOB_NAME"

	// EnvVarNamespace contains the namespace of the controller & jobs
	EnvVarNamespace = "SENSOR_NAMESPACE"

	// EnvVarConfigMap is the name of the configmap to use for the controller
	EnvVarConfigMap = "SENSOR_CONFIG_MAP"

	// EnvVarKubeConfig is the path to the Kubernetes configuration
	EnvVarKubeConfig = "KUBE_CONFIG"
)

// SENSOR JOB CONSTANTS
const (
	// WebhookServicePort is the port of the service
	WebhookServicePort = 9000

	// WebhookServiceTargetPort is the port of the targeted job
	WebhookServiceTargetPort = 9000
)
