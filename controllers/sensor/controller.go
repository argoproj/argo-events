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

package sensor

import (
	"context"
	"errors"
	"time"

	base "github.com/argoproj/argo-events"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	clientset "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// informer constants
const (
	sensorResyncPeriod   = 20 * time.Minute
	rateLimiterBaseDelay = 5 * time.Second
	rateLimiterMaxDelay  = 1000 * time.Second
)

// ControllerConfig contain the configuration settings for the controller
type ControllerConfig struct {
	// InstanceID is a label selector to limit the controller'sensor watch of sensor jobs to a specific instance.
	// If omitted, the controller watches sensors that *are not* labeled with an instance id.
	InstanceID string
	// Namespace is a label selector filter to limit controller'sensor watch to specific namespace
	Namespace string
}

// Controller listens for new sensors and hands off handling of each sensor on the queue to the operator
type Controller struct {
	// ConfigMap is the name of the config map in which to derive configuration of the controller
	ConfigMap string
	// Namespace for controller
	Namespace string
	// Config is the controller'sensor configuration
	Config ControllerConfig
	// logger to logger stuff
	logger *logrus.Logger
	// kubeConfig is the rest K8s config
	kubeConfig *rest.Config
	// k8sClient is the Kubernetes client
	k8sClient kubernetes.Interface
	// sensorClient is the client for operations on the sensor custom resource
	sensorClient clientset.Interface
	// informer for sensor resource updates
	informer cache.SharedIndexInformer
	// queue to process watched sensor resources
	queue workqueue.RateLimitingInterface
}

// NewController creates a new Controller
func NewController(rest *rest.Config, configMap, namespace string) *Controller {
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(rateLimiterBaseDelay, rateLimiterMaxDelay)
	return &Controller{
		ConfigMap:    configMap,
		Namespace:    namespace,
		kubeConfig:   rest,
		k8sClient:    kubernetes.NewForConfigOrDie(rest),
		sensorClient: clientset.NewForConfigOrDie(rest),
		queue:        workqueue.NewRateLimitingQueue(rateLimiter),
		logger:       common.NewArgoEventsLogger(),
	}
}

// processNextItem processes the sensor resource object on the queue
func (controller *Controller) processNextItem() bool {
	// Wait until there is a new item in the queue
	key, quit := controller.queue.Get()
	if quit {
		return false
	}
	defer controller.queue.Done(key)

	obj, exists, err := controller.informer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		controller.logger.WithField(common.LabelSensorName, key.(string)).WithError(err).Warnln("failed to get sensor from informer index")
		return true
	}

	if !exists {
		// this happens after sensor was deleted, but work queue still had entry in it
		return true
	}

	s, ok := obj.(*v1alpha1.Sensor)
	if !ok {
		controller.logger.WithField(common.LabelSensorName, key.(string)).WithError(err).Warnln("key in index is not a sensor")
		return true
	}

	ctx := newSensorContext(s, controller)

	err = ctx.operate()
	if err != nil {
		ctx.logger.WithError(err).WithField("sensor", s.Name).Errorln("failed to operate on the sensor object")
	}

	err = controller.handleErr(err, key)
	if err != nil {
		ctx.logger.WithError(err).Errorln("controller is unable to handle the error")
	}
	return true
}

// handleErr checks if an error happened and make sure we will retry later
// returns an error if unable to handle the error
func (controller *Controller) handleErr(err error, key interface{}) error {
	if err == nil {
		// Forget about the #AddRateLimited history of key on every successful sync
		// Ensure future updates for this key are not delayed because of outdated error history
		controller.queue.Forget(key)
		return nil
	}

	// due to the base delay of 5ms of the DefaultControllerRateLimiter
	// re-queues will happen very quickly even after a sensor pod goes down
	// we want to give the sensor pod a chance to come back up so we give a generous number of retries
	if controller.queue.NumRequeues(key) < 20 {
		// Re-enqueue the key rate limited. This key will be processed later again.
		controller.queue.AddRateLimited(key)
		return nil
	}
	return errors.New("exceeded max re-queues")
}

// Run executes the controller
func (controller *Controller) Run(ctx context.Context, threads int) {
	defer controller.queue.ShutDown()

	controller.logger.WithFields(
		map[string]interface{}{
			common.LabelInstanceID: controller.Config.InstanceID,
			common.LabelVersion:    base.GetVersion().Version,
		}).Infoln("starting the controller...")

	configMapCtrl := controller.watchControllerConfigMap()
	go configMapCtrl.Run(ctx.Done())

	var err error
	controller.informer, err = controller.newSensorInformer()
	if err != nil {
		controller.logger.WithError(err).Errorln("failed to create a new sensor controller")
	}
	go controller.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), controller.informer.HasSynced) {
		controller.logger.Panic("timed out waiting for the caches to sync for sensors")
		return
	}

	for i := 0; i < threads; i++ {
		go wait.Until(controller.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
}

func (controller *Controller) runWorker() {
	for controller.processNextItem() {
	}
}
