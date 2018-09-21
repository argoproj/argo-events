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


	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	base "github.com/argoproj/argo-events"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensorclientset "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"fmt"
	"log"
)

const (
	sensorResyncPeriod = 20 * time.Minute
)

// SensorControllerConfig contain the configuration settings for the sensor-controller
type SensorControllerConfig struct {
	// InstanceID is a label selector to limit the sensor-controller's watch of sensor jobs to a specific instance.
	// If omitted, the sensor-controller watches sensors that *are not* labeled with an instance id.
	InstanceID string

	// Namespace is a label selector filter to limit sensor-controller's watch to specific namespace
	Namespace string
}

// SensorController listens for new sensors and hands off handling of each sensor on the queue to the operator
type SensorController struct {
	// ConfigMap is the name of the config map in which to derive configuration of the contoller
	ConfigMap string
	// namespace for the config map
	ConfigMapNS string
	// Config is the sensor sensor-controller's configuration
	Config SensorControllerConfig

	// kubernetes config and apis
	kubeConfig      *rest.Config
	kubeClientset   kubernetes.Interface
	sensorClientset sensorclientset.Interface

	// sensor informer and queue
	informer cache.SharedIndexInformer
	queue    workqueue.RateLimitingInterface
}

// NewSensorController creates a new Controller
func NewSensorController(rest *rest.Config, configMap string) *SensorController {
	return &SensorController{
		ConfigMap:       configMap,
		kubeConfig:      rest,
		kubeClientset:   kubernetes.NewForConfigOrDie(rest),
		sensorClientset: sensorclientset.NewForConfigOrDie(rest),
		queue:           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

func (c *SensorController) processNextItem() bool {
	// Wait until there is a new item in the queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	obj, exists, err := c.informer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		fmt.Errorf("failed to get sensor '%s' from informer index: %+v", key, err)
		return true
	}

	if !exists {
		// this happens after sensor was deleted, but work queue still had entry in it
		return true
	}

	sensor, ok := obj.(*v1alpha1.Sensor)
	if !ok {
		fmt.Errorf("key '%s' in index is not a sensor", key)
		return true
	}

	ctx := newSensorOperationCtx(sensor, c)

	err = c.handleErr(ctx.operate(), key)
	if err != nil {
		// now let's escalate the sensor
		// the context should have the most up-to-date version
		ctx.log.Error().Err(err).Msg("escalating controller failure")
		event := ctx.GetK8Event("controller error", v1alpha1.NodePhaseError, sensor.Kind)
		_, err = common.CreateK8Event(event, ctx.controller.kubeClientset)
		if err != nil {
			ctx.log.Error().Err(err).Msg("failed to create escalation event for controller failure")
		}
	}

	return true
}

// handleErr checks if an error happened and make sure we will retry later
// returns an error if unable to handle the error
func (c *SensorController) handleErr(err error, key interface{}) error {
	if err == nil {
		// Forget about the #AddRateLimited history of key on every successful sync
		// Ensure future updates for this key are not delayed because of outdated error history
		c.queue.Forget(key)
		return nil
	}

	// due to the base delay of 5ms of the DefaultControllerRateLimiter
	// requeues will happen very quickly even after a signal pod goes down
	// we want to give the signal pod a chance to come back up so we give a genorous number of retries
	if c.queue.NumRequeues(key) < 20 {
		fmt.Errorf("Error syncing sensor '%v': %v", key, err)

		// Re-enqueue the key rate limited. This key will be processed later again.
		c.queue.AddRateLimited(key)
		return nil
	}
	return errors.New("exceeded max requeues")
}

// Run executes the sensor-controller
func (c *SensorController) Run(ctx context.Context, ssThreads, signalThreads int) {
	defer c.queue.ShutDown()

	fmt.Printf("sensor sensor-controller (version: %s) (instance: %s) starting", base.GetVersion(), c.Config.InstanceID)
	_, err := c.watchControllerConfigMap(ctx)
	if err != nil {
		fmt.Errorf("failed to register watch for sensor-controller config map: %v", err)
		return
	}

	c.informer = c.newSensorInformer()
	go c.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		log.Panicf("timed out waiting for the caches to sync")
		return
	}

	for i := 0; i < ssThreads; i++ {
		go wait.Until(c.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
}

func (c *SensorController) runWorker() {
	for c.processNextItem() {
	}
}
