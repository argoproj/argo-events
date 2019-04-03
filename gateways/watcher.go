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

package gateways

import (
	"context"
	"fmt"
	"time"

	"github.com/argoproj/argo-events/common"
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

func (gc *GatewayConfig) WatchResource(ctx context.Context, config *rest.Config, gvr schema.GroupVersionResource, namespace string, labels, fields map[string]string, resync time.Duration, resultCh chan InformerResult) error {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	client.Resource(gvr)

	options := &metav1.ListOptions{}

	if labels != nil {
		sel, err := common.LabelSelector(labels)
		if err != nil {
			return err
		}
		options.LabelSelector = sel.String()
	}

	if fields != nil {
		sel, err := common.FieldSelector(fields)
		if err != nil {
			return err
		}
		options.FieldSelector = sel.String()
	}

	tweakListOptions := func(op *metav1.ListOptions) {
		op = options
	}

	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, resync, namespace, tweakListOptions)

	informer := factory.ForResource(gvr)

	sharedInformer := informer.Informer()
	sharedInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				resultCh <- InformerResult{
					Obj:  obj,
					Type: ADD,
				}
				fmt.Println("Add ok")
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				resultCh <- InformerResult{
					Obj:    newObj,
					OldObj: oldObj,
					Type:   UPDATE,
				}
				fmt.Println("update ok")
			},
			DeleteFunc: func(obj interface{}) {
				resultCh <- InformerResult{
					Obj:  obj,
					Type: DELETE,
				}
				fmt.Println("delete ok")
			},
		},
	)
	sharedInformer.Run(ctx.Done())
	return nil
}
