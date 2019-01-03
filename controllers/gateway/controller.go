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
	"strings"
	"time"

	zlog "github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	base "github.com/argoproj/argo-events"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	clientset "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

const (
	gatewayResyncPeriod = 20 * time.Minute
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
	// ConfigMap is the name of the config map in which to derive configuration of the contoller
	ConfigMap string
	// Namespace for gateway controller
	Namespace string
	// Config is the gateway-controller gateway-controller-controller's configuration
	Config GatewayControllerConfig
	// log is the logger for a gateway
	log zlog.Logger

	// kubernetes config and apis
	kubeConfig       *rest.Config
	kubeClientset    kubernetes.Interface
	gatewayClientset clientset.Interface

	// gateway-controller informer and queue
	informer cache.SharedIndexInformer
	queue    workqueue.RateLimitingInterface
}

// NewGatewayController creates a new Controller
func NewGatewayController(rest *rest.Config, configMap, namespace string) *GatewayController {
	return &GatewayController{
		ConfigMap:        configMap,
		Namespace:        namespace,
		kubeConfig:       rest,
		log:              zlog.New(os.Stdout).With().Caller().Logger().Output(zlog.ConsoleWriter{
			Out: os.Stdout,
			FormatCaller: func(i interface{}) string {
				cwd, err := os.Getwd()
				if err == nil {
					return strings.TrimRight(cwd, "/go")
				}
				return ""
			},
		}),
		kubeClientset:    kubernetes.NewForConfigOrDie(rest),
		gatewayClientset: clientset.NewForConfigOrDie(rest),
		queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
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
		c.log.Warn().Str("gateway-controller", key.(string)).Err(err).Msg("failed to get gateway-controller '%s' from informer index")
		return true
	}

	if !exists {
		// this happens after gateway-controller was deleted, but work queue still had entry in it
		return true
	}

	gateway, ok := obj.(*v1alpha1.Gateway)
	if !ok {
		c.log.Warn().Str("gateway-controller", key.(string)).Err(err).Msg("key in index is not a gateway-controller")
		return true
	}

	ctx := newGatewayOperationCtx(gateway, c)

	err = ctx.operate()
	if err != nil {
		escalationEvent := &corev1.Event{
			Reason: err.Error(),
			Type:   string(common.EscalationEventType),
			Action: "gateway is marked as failed",
			EventTime: metav1.MicroTime{
				Time: time.Now(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace:    gateway.Namespace,
				GenerateName: gateway.Name + "-",
				Labels: map[string]string{
					common.LabelEventSeen:    "",
					common.LabelResourceName: gateway.Name,
					common.LabelEventType:    string(common.EscalationEventType),
				},
			},
			InvolvedObject: corev1.ObjectReference{
				Namespace: gateway.Namespace,
				Name:      gateway.Name,
				Kind:      gateway.Kind,
			},
			Source: corev1.EventSource{
				Component: gateway.Name,
			},
			ReportingInstance:   c.Config.InstanceID,
			ReportingController: common.DefaultGatewayControllerDeploymentName,
		}
		ctx.log.Error().Str("escalation-msg", err.Error()).Msg("escalating gateway error")
		_, err = common.CreateK8Event(escalationEvent, c.kubeClientset)
		if err != nil {
			ctx.log.Error().Err(err).Msg("failed to escalate controller error")
		}
	}

	err = c.handleErr(err, key)
	// create k8 event to escalate the error
	if err != nil {
		ctx.log.Error().Interface("error", err).Msg("gateway controller failed to handle error")
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
	// requeues will happen very quickly even after a signal pod goes down
	// we want to give the signal pod a chance to come back up so we give a genorous number of retries
	if c.queue.NumRequeues(key) < 20 {
		c.log.Error().Str("gateway-controller", key.(string)).Err(err).Msg("error syncing gateway-controller")

		// Re-enqueue the key rate limited. This key will be processed later again.
		c.queue.AddRateLimited(key)
		return nil
	}
	return errors.New("exceeded max requeues")
}

// Run executes the gateway-controller
func (c *GatewayController) Run(ctx context.Context, gwThreads, signalThreads int) {
	defer c.queue.ShutDown()
	c.log.Info().Str("version", base.GetVersion().Version).Str("instance-id", c.Config.InstanceID).Msg("starting gateway-controller")
	_, err := c.watchControllerConfigMap(ctx)
	if err != nil {
		c.log.Error().Err(err).Msg("failed to register watch for gateway-controller config map")
		return
	}

	c.informer = c.newGatewayInformer()
	go c.informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), c.informer.HasSynced) {
		c.log.Panic().Msg("timed out waiting for the caches to sync")
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
