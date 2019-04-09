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

package gateway

import (
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"

	base "github.com/argoproj/argo-events"
	"github.com/argoproj/argo-events/common"
	ccommon "github.com/argoproj/argo-events/controllers/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	clientset "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	gatewayResyncPeriod         = 20 * time.Minute
	gatewayResourceResyncPeriod = 30 * time.Minute
	rateLimiterBaseDelay        = 5 * time.Second
	rateLimiterMaxDelay         = 1000 * time.Second
)

// GatewayControllerConfig contain the configuration settings for the gateway-controller
type GatewayControllerConfig struct {
	// InstanceID is a label selector to limit the gateway-controller's watch of gateway jobs to a specific instance.
	InstanceID string

	// Namespace is a label selector filter to limit gateway-controller-controller's watch to specific namespace
	Namespace string
}

// GatewayController listens for new gateways and hands off handling of each gateway-controller on the queue to the operator
type GatewayController struct {
	// EventSource is the name of the config map in which to derive configuration of the contoller
	ConfigMap string
	// Namespace for gateway controller
	Namespace string
	// Config is the gateway-controller gateway-controller-controller's configuration
	Config GatewayControllerConfig
	// log is the logger for a gateway
	log *logrus.Logger

	// kubernetes config and apis
	kubeConfig       *rest.Config
	kubeClientset    kubernetes.Interface
	gatewayClientset clientset.Interface

	// gateway-controller informer and queue
	podInformer informersv1.PodInformer
	svcInformer informersv1.ServiceInformer
	informer    cache.SharedIndexInformer
	queue       workqueue.RateLimitingInterface
}

// NewGatewayController creates a new Controller
func NewGatewayController(rest *rest.Config, configMap, namespace string) *GatewayController {
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(rateLimiterBaseDelay, rateLimiterMaxDelay)
	return &GatewayController{
		ConfigMap:        configMap,
		Namespace:        namespace,
		kubeConfig:       rest,
		log:              common.NewArgoEventsLogger(),
		kubeClientset:    kubernetes.NewForConfigOrDie(rest),
		gatewayClientset: clientset.NewForConfigOrDie(rest),
		queue:            workqueue.NewRateLimitingQueue(rateLimiter),
	}
}

func (c *GatewayController) processNextItem() bool {
	// Wait until there is a new item in the queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	obj, exists, err := c.informer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		c.log.WithField(common.LabelGatewayName, key.(string)).WithError(err).Warn("failed to get gateway from informer index")
		return true
	}

	if !exists {
		// this happens after gateway-controller was deleted, but work queue still had entry in it
		return true
	}

	gw, ok := obj.(*v1alpha1.Gateway)
	if !ok {
		c.log.WithField(common.LabelGatewayName, key.(string)).WithError(err).Warn("key in index is not a gateway")
		return true
	}

	ctx := newGatewayOperationCtx(gw, c)

	err = ctx.operate()
	if err != nil {
		if err := common.GenerateK8sEvent(c.kubeClientset,
			fmt.Sprintf("controller failed to operate on gateway %s", gw.Name),
			common.StateChangeEventType,
			"controller operation failed",
			gw.Name,
			gw.Namespace,
			c.Config.InstanceID,
			gw.Kind,
			map[string]string{
				common.LabelGatewayName: gw.Name,
				common.LabelEventType:   string(common.EscalationEventType),
			},
		); err != nil {
			ctx.log.WithError(err).Error("failed to create K8s event to escalate controller operation failure")
		}
	}

	err = c.handleErr(err, key)
	// create k8 event to escalate the error
	if err != nil {
		ctx.log.WithError(err).Error("gateway controller failed to handle error")
	}
	return true
}

// handleErr checks if an error happened and make sure we will retry later
// returns an error if unable to handle the error
func (c *GatewayController) handleErr(err error, key interface{}) error {
	if err == nil {
		// Forget about the #AddRateLimited history of key on every successful sync
		// Ensure future updates for this key are not delayed because of outdated error history
		c.queue.Forget(key)
		return nil
	}

	// due to the base delay of 5ms of the DefaultControllerRateLimiter
	// requeues will happen very quickly even after a gateway pod goes down
	// we want to give the event pod a chance to come back up so we give a generous number of retries
	if c.queue.NumRequeues(key) < 20 {
		c.log.WithField(common.LabelGatewayName, key.(string)).WithError(err).Error("error syncing gateway")

		// Re-enqueue the key rate limited. This key will be processed later again.
		c.queue.AddRateLimited(key)
		return nil
	}
	return errors.New("exceeded max requeues")
}

// Run executes the gateway-controller
func (c *GatewayController) Run(ctx context.Context, gwThreads, eventThreads int) {
	defer c.queue.ShutDown()
	c.log.WithFields(
		map[string]interface{}{
			common.LabelInstanceID: c.Config.InstanceID,
			common.LabelVersion:    base.GetVersion().Version,
		}).Info("starting gateway-controller")
	_, err := c.watchControllerConfigMap(ctx)
	if err != nil {
		c.log.WithError(err).Error("failed to register watch for gateway-controller config map")
		return
	}

	c.informer = c.newGatewayInformer()
	go c.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		c.log.Panicf("timed out waiting for the caches to sync for gateways")
		return
	}

	listOptionsFunc := func(options *metav1.ListOptions) {
		labelSelector := labels.NewSelector().Add(c.instanceIDReq())
		options.LabelSelector = labelSelector.String()
	}
	factory := ccommon.ArgoEventInformerFactory{
		OwnerGroupVersionKind: v1alpha1.SchemaGroupVersionKind,
		OwnerInformer:         c.informer,
		SharedInformerFactory: informers.NewFilteredSharedInformerFactory(c.kubeClientset, gatewayResourceResyncPeriod, c.Config.Namespace, listOptionsFunc),
		Queue: c.queue,
	}

	c.podInformer = factory.NewPodInformer()
	go c.podInformer.Informer().Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.podInformer.Informer().HasSynced) {
		c.log.Panic("timed out waiting for the caches to sync for gateway pods")
		return
	}

	c.svcInformer = factory.NewServiceInformer()
	go c.svcInformer.Informer().Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.svcInformer.Informer().HasSynced) {
		c.log.Panic("timed out waiting for the caches to sync for gateway services")
		return
	}

	for i := 0; i < gwThreads; i++ {
		go wait.Until(c.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
}

func (c *GatewayController) runWorker() {
	for c.processNextItem() {
	}
}
