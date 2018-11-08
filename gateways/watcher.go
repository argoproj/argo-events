package gateways

import (
	"fmt"
	"github.com/argoproj/argo-events/common"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"context"
)

// WatchGatewayEvents watches events generated in namespace
func (gc *GatewayConfig) WatchGatewayEvents(ctx context.Context) (cache.Controller, error) {
	source := gc.newEventWatcher()
	_, controller := cache.NewInformer(
		source,
		&corev1.Event{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if newEvent, ok := obj.(*corev1.Event); ok {
					if gc.filterEvent(newEvent) {
						gc.Log.Info().Msg("detected new k8 Event. Updating gateway resource.")
						err := gc.updateGatewayResource(newEvent)
						if err != nil {
							gc.Log.Error().Err(err).Msg("update of gateway resource failed")
						}
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if event, ok := new.(*corev1.Event); ok {
					if gc.filterEvent(event) {
						gc.Log.Info().Msg("detected k8 Event update. Updating gateway resource.")
						err := gc.updateGatewayResource(event)
						if err != nil {
							gc.Log.Error().Err(err).Msg("update of gateway resource failed")
						}
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// newEventWatcher creates a new event watcher.
func (gc *GatewayConfig) newEventWatcher() *cache.ListWatch {
	x := gc.Clientset.CoreV1().RESTClient()
	resource := "events"
	labelSelector := fields.ParseSelectorOrDie(fmt.Sprintf("%s=%s", common.LabelGatewayName, gc.Name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = labelSelector.String()
		req := x.Get().
			Namespace(gc.gw.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.LabelSelector = labelSelector.String()
		options.Watch = true
		req := x.Get().
			Namespace(gc.gw.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// filters unwanted events
func (gc *GatewayConfig) filterEvent(event *corev1.Event) bool {
	if event.Source.Component == gc.gw.Name &&
		event.ObjectMeta.Labels[common.LabelEventSeen] == "" &&
		event.ReportingInstance == gc.controllerInstanceID &&
		event.ReportingController == gc.gw.Name {
		gc.Log.Info().Str("event-name", event.ObjectMeta.Name).Msg("processing gateway k8 event")
		return true
	}
	return false
}

// WatchGatewayConfigMap watches change in configuration for the gateway
func (gc *GatewayConfig) WatchGatewayConfigMap(ctx context.Context, executor ConfigExecutor) (cache.Controller, error) {
	source := gc.newConfigMapWatch(gc.configName)
	_, controller := cache.NewInformer(
		source,
		&corev1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if newCm, ok := obj.(*corev1.ConfigMap); ok {
					gc.Log.Info().Str("config-map", gc.configName).Msg("detected ConfigMap addition. Updating the controller run config.")
					err := gc.manageConfigurations(executor, newCm)
					if err != nil {
						gc.Log.Error().Err(err).Msg("add config failed")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if cm, ok := new.(*corev1.ConfigMap); ok {
					gc.Log.Info().Msg("detected ConfigMap update. Updating the controller run config.")
					err := gc.manageConfigurations(executor, cm)
					if err != nil {
						gc.Log.Error().Err(err).Msg("update config failed")
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// newConfigMapWatch creates a new configmap watcher
func (gc *GatewayConfig) newConfigMapWatch(name string) *cache.ListWatch {
	x := gc.Clientset.CoreV1().RESTClient()
	resource := "configmaps"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gc.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(gc.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}
