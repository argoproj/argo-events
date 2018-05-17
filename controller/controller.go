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
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/blackrock/axis"
	"github.com/blackrock/axis/common"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
	sensorclientset "github.com/blackrock/axis/pkg/client/clientset/versioned"
)

const (
	sensorResyncPeriod = 20 * time.Minute
)

// SensorControllerConfig contain the configuration settings for the sensor controller
type SensorControllerConfig struct {
	// ExecutorImage is the name of the image to run for the container inside the sensor executor jobs
	ExecutorImage string `json:"executorImage"`

	// InstanceID is a label selector to limit the controller's watch of sensor jobs to a specific instance.
	// If omitted, the controller watches sensors that *are not* labeled with an instance id.
	InstanceID string `json:"instanceID,omitempty"`

	// Namespace is a label selector filter to limit controller's watch to specific namespace
	Namespace string `json:"namespace"`

	// The service account to attach to the sensor executor pods
	ServiceAccount string `json:"serviceAccount"`

	// ExecutorResources specifies the resource requirements that will be used for the sensor executor pods
	ExecutorResources *corev1.ResourceRequirements `json:"executorResources,omitempty"`
}

// SensorController listens for new sensors and hands off handling of each sensor on the queue to the operator
type SensorController struct {
	// ConfigMap is the name of the config map in which to derive configuration of the contoller
	ConfigMap string
	// namespace for the config map
	ConfigMapNS string
	// Config is the sensor controller's configuration
	Config SensorControllerConfig

	kubeConfig      *rest.Config
	kubeClientset   kubernetes.Interface
	sensorClientset sensorclientset.Interface

	ssQueue     workqueue.RateLimitingInterface
	podQueue    workqueue.RateLimitingInterface
	ssInformer  cache.SharedIndexInformer
	podInformer cache.SharedIndexInformer

	log *zap.SugaredLogger
}

// NewSensorController creates a new Controller
func NewSensorController(rest *rest.Config, kubeClient kubernetes.Interface, sensorClient sensorclientset.Interface, log *zap.SugaredLogger, configMap string) *SensorController {
	return &SensorController{
		kubeConfig:      rest,
		kubeClientset:   kubeClient,
		sensorClientset: sensorClient,
		ConfigMap:       configMap,
		ssQueue:         workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		podQueue:        workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		log:             log,
	}
}

func (c *SensorController) processNextItem() bool {
	// Wait until there is a new item in the queue
	key, quit := c.ssQueue.Get()
	if quit {
		return false
	}
	defer c.ssQueue.Done(key)

	obj, exists, err := c.ssInformer.GetIndexer().GetByKey(key.(string))
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
		c.ssQueue.Forget(key)
		return
	}

	if c.ssQueue.NumRequeues(key) < 3 {
		c.log.Errorf("Error syncing sensor %v: %v", key, err)

		// Re-enqueue the key rate limited. This key will be processed later again.
		c.ssQueue.AddRateLimited(key)
		return
	}
}

// Run executes the controller
func (c *SensorController) Run(ctx context.Context, ssThreads, jobThreads int) {
	defer c.ssQueue.ShutDown()
	defer c.podQueue.ShutDown()

	c.log.Infof("sensor controller (version: %s) (instance: %s) starting", axis.GetVersion(), c.Config.InstanceID)
	_, err := c.watchControllerConfigMap(ctx)
	if err != nil {
		c.log.Errorf("failed to register watch for controller config map: %v", err)
		return
	}

	c.ssInformer = c.newSensorInformer()
	c.podInformer = c.newPodInformer()
	go c.ssInformer.Run(ctx.Done())
	go c.podInformer.Run(ctx.Done())

	for _, informer := range []cache.SharedIndexInformer{c.ssInformer, c.podInformer} {
		if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
			c.log.Panicf("timed out waiting for the caches to sync")
			return
		}
	}

	for i := 0; i < ssThreads; i++ {
		go wait.Until(c.runWorker, time.Second, ctx.Done())
	}
	for i := 0; i < jobThreads; i++ {
		go wait.Until(c.podWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
}

func (c *SensorController) runWorker() {
	for c.processNextItem() {
	}
}

func (c *SensorController) podWorker() {
	for c.processNextJob() {
	}
}

func (c *SensorController) processNextJob() bool {
	key, quit := c.podQueue.Get()
	if quit {
		return false
	}
	defer c.podQueue.Done(key)

	obj, exists, err := c.podInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		c.log.Errorf("failed to get pod '%s' from informer index: %+v", key, err)
		return true
	}
	if !exists {
		// we can get here if job was queued into the job workqueue,
		// but it was either deleted or labeled completed by the time
		// we dequeued it.
		return true
	}
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		c.log.Warnf("key '%s' in index is not a pod", key)
		return true
	}
	if pod.Labels == nil {
		c.log.Warnf("pod '%s' did not have labels", key)
		return true
	}

	jobName, ok := pod.Labels[common.LabelJobName]
	if !ok {
		c.log.Warnf("pod unrelated to any sensor: %s", pod.ObjectMeta.Name)
	}

	// TODO: currently we requeue the sensor on *any* job updates
	// But this can be made smarter to only requeue on changes we care about
	c.ssQueue.Add(pod.ObjectMeta.Namespace + "/" + common.ParseJobPrefix(jobName))
	return true
}
