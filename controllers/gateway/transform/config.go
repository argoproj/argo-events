package transform

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"strings"
)

// WatchGatewayTransformerConfigMap watches gateway transformer configmap for any changes.
func (t *tOperationCtx) WatchGatewayTransformerConfigMap(ctx context.Context, name string) (cache.Controller, error) {
	source := t.newStoreConfigMapWatch(name)
	_, controller := cache.NewInformer(
		source,
		&apiv1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*apiv1.ConfigMap); ok {
					t.log.Info().Str("config-map", name).Msg("detected ConfigMap update. Updating the controller config.")
					err := t.updateConfig(cm)
					if err != nil {
						t.log.Error().Err(err).Msg("update of config failed")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if newCm, ok := new.(*apiv1.ConfigMap); ok {
					t.log.Info().Msg("detected ConfigMap update. Updating the controller config.")
					err := t.updateConfig(newCm)
					if err != nil {
						t.log.Error().Err(err).Msg("update of config failed")
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

// newStoreConfigMapWatch returns a new configmap watcher
func (t *tOperationCtx) newStoreConfigMapWatch(name string) *cache.ListWatch {
	x := t.kubeClientset.CoreV1().RESTClient()
	resource := "configmaps"
	fieldSelector := fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", name))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(t.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.FieldSelector = fieldSelector.String()
		req := x.Get().
			Namespace(t.Namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

// updateConfig updates the transformer configmap whenever there is an update.
func (t *tOperationCtx) updateConfig(cm *apiv1.ConfigMap) error {
	// it is the type of gateway
	eventType, ok := cm.Data[common.EventType]
	if !ok {
		return fmt.Errorf("configMap '%s' does not have key '%s'", cm.Name, common.EventType)
	}

	// version of cloudevents
	eventTypeVersion, ok := cm.Data[common.EventTypeVersion]
	if !ok {
		return fmt.Errorf("configMap '%s' does not have key '%s'", cm.Name, common.EventTypeVersion)
	}

	// name of the gateway
	eventSource, ok := cm.Data[common.EventSource]
	if !ok {
		return fmt.Errorf("configMap '%s' does not have key '%s'", cm.Name, common.EventSource)
	}

	// parse sensor watchers and gateway watchers if any
	var sensorWatchers []v1alpha1.SensorNotificationWatcher
	var gatewayWatchers []v1alpha1.GatewayNotificationWatcher

	sensorWatchersStr := cm.Data[common.SensorWatchers]
	gatewayWatchersStr := cm.Data[common.GatewayWatchers]

	t.log.Info().Interface("sensors", sensorWatchersStr).Msg("sensor watchers")
	t.log.Info().Interface("gateways", gatewayWatchersStr).Msg("gateway watchers")

	// parse sensor watchers
	if sensorWatchersStr != "" {
		for _, sensorWatcherStr := range strings.Split(sensorWatchersStr, ",") {
			var sensorWatcher v1alpha1.SensorNotificationWatcher
			err := yaml.Unmarshal([]byte(sensorWatcherStr), &sensorWatcher)
			if err != nil {
				panic(fmt.Errorf("failed to parse sensor watcher string. Err: %+v", err))
			}
			sensorWatchers = append(sensorWatchers, sensorWatcher)
		}
	}

	// parse gateway watchers
	if gatewayWatchersStr != "" {
		for _, gatewayWatcherStr := range strings.Split(gatewayWatchersStr, ",") {
			var gatewayWatcher v1alpha1.GatewayNotificationWatcher
			err := yaml.Unmarshal([]byte(gatewayWatcherStr), &gatewayWatcher)
			if err != nil {
				panic(fmt.Errorf("failed to parse gateway watcher string. Err: %+v", err))
			}
			gatewayWatchers = append(gatewayWatchers, gatewayWatcher)
		}
	}

	t.Config = &tConfig{
		EventType:        eventType,
		EventTypeVersion: eventTypeVersion,
		Gateways:         gatewayWatchers,
		Sensors:          sensorWatchers,
		EventSource:      eventSource,
	}
	return nil
}
