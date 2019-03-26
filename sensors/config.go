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
	"net/http"

	"github.com/nats-io/go-nats"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	clientset "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	snats "github.com/nats-io/go-nats-streaming"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// sensorExecutionCtx contains execution context for sensor
type sensorExecutionCtx struct {
	// sensorClient is the client for sensor
	sensorClient clientset.Interface
	// kubeClient is the kubernetes client
	kubeClient kubernetes.Interface
	// ClientPool manages a pool of dynamic clients.
	clientPool dynamic.ClientPool
	// DiscoveryClient implements the functions that discover server-supported API groups, versions and resources.
	discoveryClient discovery.DiscoveryInterface
	// sensor object
	sensor *v1alpha1.Sensor
	// http server which exposes the sensor to gateway/s
	server *http.Server
	// logger for the sensor
	log *common.ArgoEventsLogger
	// queue is internal queue to manage incoming events
	queue chan *updateNotification
	// controllerInstanceID is the instance ID of sensor controller processing this sensor
	controllerInstanceID string
	// updated indicates update to sensor resource
	updated bool
	// nconn is the nats connection
	nconn natsconn
}

type natsconn struct {
	// standard connection
	standard *nats.Conn
	// streaming connection
	stream snats.Conn
}

// updateNotification is servers as a notification message that can be used to update event dependency's state or the sensor resource
type updateNotification struct {
	event            *apicommon.Event
	eventDependency  *v1alpha1.EventDependency
	writer           http.ResponseWriter
	sensor           *v1alpha1.Sensor
	notificationType v1alpha1.NotificationType
}

// NewSensorExecutionCtx returns a new sensor execution context.
func NewSensorExecutionCtx(sensorClient clientset.Interface, kubeClient kubernetes.Interface,
	clientPool dynamic.ClientPool, discoveryClient discovery.DiscoveryInterface,
	sensor *v1alpha1.Sensor, controllerInstanceID string) *sensorExecutionCtx {
	return &sensorExecutionCtx{
		sensorClient:         sensorClient,
		kubeClient:           kubeClient,
		clientPool:           clientPool,
		discoveryClient:      discoveryClient,
		sensor:               sensor,
		log:                  common.NewArgoEventsLogger().WithSensorName(sensor.Name),
		queue:                make(chan *updateNotification),
		controllerInstanceID: controllerInstanceID,
	}
}
