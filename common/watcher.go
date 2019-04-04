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

package common

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type InformerAction string

const (
	ADD    InformerAction = "ADD"
	UPDATE InformerAction = "UPDATE"
	DELETE InformerAction = "DELETE"
)

type InformerResult struct {
	Obj    interface{}
	OldObj interface{}
	Type   InformerAction
}

// WatchResource instantiates a generic informer that can watch any standard or custom resource
// e.g. Argo Workflow has "argoproj.io" as group, "v1alpha1" as version and "workflows" as resource
// Make sure to pass plural for resource name.
// To return from the method and stop the informer, close the stop channel.
func WatchResource(ctx context.Context, client dynamic.Interface, group, version, resource, namespace string, labels, fields map[string]string, resync time.Duration, resultCh chan InformerResult) error {
	options := &metav1.ListOptions{}

	if labels != nil {
		sel, err := LabelSelector(labels)
		if err != nil {
			return err
		}
		options.LabelSelector = sel.String()
	}

	if fields != nil {
		sel, err := FieldSelector(fields)
		if err != nil {
			return err
		}
		options.FieldSelector = sel.String()
	}

	tweakListOptions := func(op *metav1.ListOptions) {
		op = options
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, resync, namespace, tweakListOptions)

	informer := factory.ForResource(schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	})

	sharedInformer := informer.Informer()
	sharedInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				resultCh <- InformerResult{
					Obj:  obj,
					Type: ADD,
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				resultCh <- InformerResult{
					Obj:    newObj,
					OldObj: oldObj,
					Type:   UPDATE,
				}
			},
			DeleteFunc: func(obj interface{}) {
				resultCh <- InformerResult{
					Obj:  obj,
					Type: DELETE,
				}
			},
		},
	)

	sharedInformer.Run(ctx.Done())
	return nil
}

// GetNamespaceableResourceClient returns a generic client which can be used for any standard or custom resource.
// e.g. Argo Workflow has "argoproj.io" as group, "v1alpha1" as version and "workflows" as resource
// Make sure to pass plural for resource name.
func GetNamespaceableResourceClient(config *rest.Config, gvr schema.GroupVersionResource, namespace string) (dynamic.NamespaceableResourceInterface, error) {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	nri := client.Resource(gvr)

	return nri, nil
}
