package common

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// ArgoEventInformerFactory holds values to create SharedInformerFactory of argo-events
type ArgoEventInformerFactory struct {
	OwnerGroupVersionKind schema.GroupVersionKind
	OwnerInformer         cache.SharedIndexInformer
	informers.SharedInformerFactory
	Queue workqueue.RateLimitingInterface
}

// NewPodInformer returns a PodInformer of argo-events
func (c *ArgoEventInformerFactory) NewPodInformer() informersv1.PodInformer {
	podInformer := c.SharedInformerFactory.Core().V1().Pods()
	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				gw, err := c.getOwner(obj)
				if err != nil {
					return
				}
				key, err := cache.MetaNamespaceKeyFunc(gw)
				if err == nil {
					c.Queue.Add(key)
				}
			},
		},
	)
	return podInformer
}

// NewServiceInformer returns a ServiceInformer of argo-events
func (c *ArgoEventInformerFactory) NewServiceInformer() informersv1.ServiceInformer {
	svcInformer := c.SharedInformerFactory.Core().V1().Services()
	svcInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				obj, err := c.getOwner(obj)
				if err != nil {
					return
				}
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					c.Queue.Add(key)
				}
			},
		},
	)
	return svcInformer
}

func (c *ArgoEventInformerFactory) getOwner(obj interface{}) (interface{}, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	for _, owner := range m.GetOwnerReferences() {

		if owner.APIVersion == c.OwnerGroupVersionKind.GroupVersion().String() &&
			owner.Kind == c.OwnerGroupVersionKind.Kind {
			key := owner.Name
			if len(m.GetNamespace()) > 0 {
				key = m.GetNamespace() + "/" + key
			}
			obj, exists, err := c.OwnerInformer.GetIndexer().GetByKey(key)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, fmt.Errorf("failed to get object from cache")
			}
			return obj, nil
		}
	}
	return nil, fmt.Errorf("no owner found")
}
