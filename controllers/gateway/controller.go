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
	"time"

	base "github.com/argoproj/argo-events"
	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	clientset "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	gatewayResyncPeriod  = 10 * time.Minute
	rateLimiterBaseDelay = 5 * time.Second
	rateLimiterMaxDelay  = 1000 * time.Second
)

// ControllerConfig contain the configuration settings for the controller
type ControllerConfig struct {
	// InstanceID is a label selector to limit the controller's watch of gateway to a specific instance.
	InstanceID string
	// Namespace is a label selector filter to limit controller's watch to specific namespace
	Namespace string
}

// Controller listens for new gateways and hands off handling of each gateway controller on the queue to the operator
type Controller struct {
	// ConfigMap is the name of the config map in which to derive configuration of the controller
	ConfigMap string
	// Namespace for controller
	Namespace string
	// Config is the controller's configuration
	Config ControllerConfig
	// clientImage is the image of gatewayClient
	clientImage string
	// gatewayImages is the map to store different types of gateway adapter images
	gatewayImages map[apicommon.EventSourceType]string
	// logger to logger stuff
	logger *logrus.Logger
	// K8s rest config
	kubeConfig *rest.Config
	// k8sClient is Kubernetes client
	k8sClient kubernetes.Interface
	// gatewayClient is the Argo-Events gateway resource client
	gatewayClient clientset.Interface
	// gateway-controller informer and queue
	informer cache.SharedIndexInformer
	queue    workqueue.RateLimitingInterface
}

// NewGatewayController creates a new controller
func NewGatewayController(rest *rest.Config, configMap, namespace, clientImage string, gatewayImages map[apicommon.EventSourceType]string) *Controller {
	rateLimiter := workqueue.NewItemExponentialFailureRateLimiter(rateLimiterBaseDelay, rateLimiterMaxDelay)
	return &Controller{
		ConfigMap:     configMap,
		Namespace:     namespace,
		clientImage:   clientImage,
		gatewayImages: gatewayImages,
		kubeConfig:    rest,
		logger:        common.NewArgoEventsLogger(),
		k8sClient:     kubernetes.NewForConfigOrDie(rest),
		gatewayClient: clientset.NewForConfigOrDie(rest),
		queue:         workqueue.NewRateLimitingQueue(rateLimiter),
	}
}

// processNextItem processes a gateway resource on the controller's queue
func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	obj, exists, err := c.informer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		c.logger.WithField(common.LabelResourceName, key.(string)).WithError(err).Warnln("failed to get gateway from informer index")
		return true
	}

	if !exists {
		// this happens after controller was deleted, but work queue still had entry in it
		return true
	}

	gw, ok := obj.(*v1alpha1.Gateway)
	if !ok {
		c.logger.WithField(common.LabelResourceName, key.(string)).WithError(err).Warnln("key in index is not a gateway")
		return true
	}

	ctx := newGatewayContext(gw, c)

	err = ctx.operate()
	if err != nil {
		ctx.logger.WithField("gateway", gw.Name).WithError(err).Errorln("failed to operate on the gateway object")
	}

	err = c.handleErr(err, key)
	// create k8 event to escalate the error
	if err != nil {
		ctx.logger.WithError(err).Errorln("controller failed to handle error")
	}
	return true
}

// handleErr checks if an error happened and make sure we will retry later
// returns an error if unable to handle the error
func (c *Controller) handleErr(err error, key interface{}) error {
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
		c.logger.WithField(common.LabelResourceName, key.(string)).WithError(err).Errorln("error syncing gateway")

		// Re-enqueue the key rate limited. This key will be processed later again.
		c.queue.AddRateLimited(key)
		return nil
	}
	return errors.New("exceeded max requeues")
}

// Run processes the gateway resources on the controller's queue
func (c *Controller) Run(ctx context.Context, threads int) {
	defer c.queue.ShutDown()
	c.logger.WithFields(
		map[string]interface{}{
			common.LabelInstanceID: c.Config.InstanceID,
			common.LabelVersion:    base.GetVersion().Version,
		}).Infoln("starting controller")

	configMapCtrl := c.watchControllerConfigMap()
	go configMapCtrl.Run(ctx.Done())

	var err error
	c.informer, err = c.newGatewayInformer()
	if err != nil {
		panic(err)
	}

	go c.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		c.logger.Errorln("timed out waiting for the caches to sync for gateways")
		return
	}

	for i := 0; i < threads; i++ {
		go wait.Until(c.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}
