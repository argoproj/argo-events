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
	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/sirupsen/logrus"
	"net/http"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	clientset "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// sensorContext contains execution context for sensor
type sensorContext struct {
	// sensorClient is the client for sensor
	sensorClient clientset.Interface
	// kubeClient is the kubernetes client
	kubeClient kubernetes.Interface
	// ClientPool manages a pool of dynamic clients.
	dynamicClient dynamic.Interface
	// sensor object
	sensor *v1alpha1.Sensor
	// http server which exposes the sensor to gateway/s
	server *http.Server
	// logger for the sensor
	logger *logrus.Logger
	// notificationQueue is internal notificationQueue to manage incoming events
	notificationQueue chan *notification
	// controllerInstanceID is the instance ID of sensor controller processing this sensor
	controllerInstanceID string
	// updated indicates update to sensor resource
	updated bool
}

// notification is servers as a notification message that can be used to update event dependency's state or the sensor resource
type notification struct {
	event            *cloudevents.Event
	eventDependency  *v1alpha1.EventDependency
	sensor           *v1alpha1.Sensor
	notificationType v1alpha1.NotificationType
}

// NewSensorExecutionCtx returns a new sensor execution context.
func NewSensorExecutionCtx(sensorClient clientset.Interface, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, sensor *v1alpha1.Sensor, controllerInstanceID string) *sensorContext {
	return &sensorContext{
		sensorClient:         sensorClient,
		kubeClient:           kubeClient,
		dynamicClient:        dynamicClient,
		sensor:               sensor,
		logger:               common.NewArgoEventsLogger().WithField(common.LabelSensorName, sensor.Name).Logger,
		notificationQueue:    make(chan *notification),
		controllerInstanceID: controllerInstanceID,
	}
}
