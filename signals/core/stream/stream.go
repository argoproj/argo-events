package stream

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"fmt"
	"context"
	"github.com/rs/zerolog"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GatewayConfig provides a generic configuration for a gateway
type GatewayConfig struct {
	// Log provides fast and simple logger dedicated to JSON output
	Log zerolog.Logger
	// Clientset is client for kubernetes API
	Clientset *kubernetes.Clientset
	// Namespace is namespace for the gateway to run inside
	Namespace string
}

// Stream describes a queue stream resource
type Stream struct {
	// Type of the stream resource
	Type string `json:"type" protobuf:"bytes,1,opt,name=type"`

	// URL is the exposed endpoint for client connections to this service
	URL string `json:"url" protobuf:"bytes,2,opt,name=url"`

	// Attributes contains additional fields specific to each service implementation
	Attributes map[string]string `json:"attributes,omitempty" protobuf:"bytes,3,rep,name=attributes"`
}

type GatewayExecutor interface {
	RunGateway(cm *apiv1.ConfigMap) error
}

func (gc *GatewayConfig) WatchGatewayConfigMap(gtEx GatewayExecutor, ctx context.Context, name string) (cache.Controller, error) {
	source := gc.newStoreConfigMapWatch(name)
	_, controller := cache.NewInformer(
		source,
		&apiv1.ConfigMap{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if cm, ok := obj.(*apiv1.ConfigMap); ok {
					gc.Log.Info().Str("config-map", name).Msg("detected ConfigMap update. Updating the controller config.")
					err := (gtEx).RunGateway(cm)
					if err != nil {
						gc.Log.Error().Err(err).Msg("update of config failed")
					}
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if newCm, ok := new.(*apiv1.ConfigMap); ok {
					gc.Log.Info().Msg("detected ConfigMap update. Updating the controller config.")
					err := (gtEx).RunGateway(newCm)
					if err != nil {
						gc.Log.Error().Err(err).Msg("update of config failed")
					}
				}
			},
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (gc *GatewayConfig) newStoreConfigMapWatch(name string) *cache.ListWatch {
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
