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

package controller

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-events"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensorclientset "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-events/shared"
)

const (
	sensorResyncPeriod      = 20 * time.Minute
	pluginHealthCheckPeriod = 30 * time.Second
)

// SensorControllerConfig contain the configuration settings for the sensor controller
type SensorControllerConfig struct {
	// InstanceID is a label selector to limit the controller's watch of sensor jobs to a specific instance.
	// If omitted, the controller watches sensors that *are not* labeled with an instance id.
	InstanceID string `json:"instanceID,omitempty"`

	// Namespace is a label selector filter to limit controller's watch to specific namespace
	Namespace string `json:"namespace"`
}

// SensorController listens for new sensors and hands off handling of each sensor on the queue to the operator
type SensorController struct {
	// ConfigMap is the name of the config map in which to derive configuration of the contoller
	ConfigMap string
	// namespace for the config map
	ConfigMapNS string
	// Config is the sensor controller's configuration
	Config SensorControllerConfig

	// kubernetes config and apis
	kubeConfig      *rest.Config
	kubeClientset   kubernetes.Interface
	sensorClientset sensorclientset.Interface

	// sensor informer and queue
	informer cache.SharedIndexInformer
	queue    workqueue.RateLimitingInterface

	// plugins
	signalProto plugin.ClientProtocol

	// inventories
	signalMu sync.Mutex
	signals  map[string]shared.Signaler

	log *zap.SugaredLogger
}

// NewSensorController creates a new Controller
func NewSensorController(rest *rest.Config, configMap string, signalProto plugin.ClientProtocol, log *zap.SugaredLogger) *SensorController {
	return &SensorController{
		ConfigMap:       configMap,
		kubeConfig:      rest,
		kubeClientset:   kubernetes.NewForConfigOrDie(rest),
		sensorClientset: sensorclientset.NewForConfigOrDie(rest),
		queue:           workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		signalProto:     signalProto,
		signals:         make(map[string]shared.Signaler),
		log:             log,
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
		c.log.Warnf("failed to get sensor '%s' from informer index: %+v", key, err)
		return true
	}

	if !exists {
		// this happens after sensor was deleted, but work queue still had entry in it
		return true
	}

	sensor, ok := obj.(*v1alpha1.Sensor)
	if !ok {
		c.log.Warnf("key '%s' in index is not a sensor", key)
		return true
	}

	ctx := newSensorOperationCtx(sensor, c)
	err = ctx.operate()

	c.handleErr(err, key)

	return true
}

// handleErr checks if an error happened and make sure we will retry later - do we want to retry later? probably
func (c *SensorController) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of key on every successful synch
		// Ensure future updates for this key are not delayed because of outdated error history
		c.queue.Forget(key)
		return
	}

	if c.queue.NumRequeues(key) < 3 {
		c.log.Errorf("Error syncing sensor %v: %v", key, err)

		// Re-enqueue the key rate limited. This key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}
}

// Run executes the controller
func (c *SensorController) Run(ctx context.Context, ssThreads, signalThreads int) {
	defer c.queue.ShutDown()

	c.log.Infof("sensor controller (version: %s) (instance: %s) starting", axis.GetVersion(), c.Config.InstanceID)
	_, err := c.watchControllerConfigMap(ctx)
	if err != nil {
		c.log.Errorf("failed to register watch for controller config map: %v", err)
		return
	}

	c.informer = c.newSensorInformer()
	go c.informer.Run(ctx.Done())
	go c.monitorPlugins(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		c.log.Panicf("timed out waiting for the caches to sync")
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

// monitors the plugins to ensure they are all still up & running
// if a ping fails, we exit the program so k8s can restart the controller
// and re-initialize a connection to the plugin processes
func (c *SensorController) monitorPlugins(done <-chan struct{}) {
	timer := time.NewTimer(pluginHealthCheckPeriod)
	for {
		select {
		case <-timer.C:
			err := c.signalProto.Ping()
			if err != nil {
				c.log.Fatalf("signal plugin client connection failed")
			}
		case <-done:
			timer.Stop()
			return
		}
	}
}
