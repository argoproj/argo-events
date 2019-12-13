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

package sensors

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	clientset "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// SensorContext contains execution context for Sensor
type SensorContext struct {
	// SensorClient is the client for Sensor
	SensorClient clientset.Interface
	// KubeClient is the kubernetes client
	KubeClient kubernetes.Interface
	// ClientPool manages a pool of dynamic clients.
	DynamicClient dynamic.Interface
	// Sensor object
	Sensor *v1alpha1.Sensor
	// Logger for the Sensor
	Logger *logrus.Logger
	// NotificationQueue is internal NotificationQueue to manage incoming events
	NotificationQueue chan *Notification
	// ControllerInstanceID is the instance ID of Sensor controller processing this Sensor
	ControllerInstanceID string
	// Updated indicates update to Sensor resource
	Updated bool
}

// Notification is servers as a Notification message that can be used to update Event dependency's state or the Sensor resource
type Notification struct {
	Event            *cloudevents.Event
	EventDependency  *v1alpha1.EventDependency
	NotificationType v1alpha1.NotificationType
}

// NewSensorExecutionCtx returns a new Sensor execution context.
func NewSensorExecutionCtx(sensorClient clientset.Interface, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, sensor *v1alpha1.Sensor, controllerInstanceID string) *SensorContext {
	return &SensorContext{
		SensorClient:         sensorClient,
		KubeClient:           kubeClient,
		DynamicClient:        dynamicClient,
		Sensor:               sensor,
		Logger:               common.NewArgoEventsLogger().WithField(common.LabelSensorName, sensor.Name).Logger,
		NotificationQueue:    make(chan *Notification),
		ControllerInstanceID: controllerInstanceID,
	}
}
