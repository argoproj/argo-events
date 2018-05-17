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
	"fmt"

	"github.com/nats-io/go-nats"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	"github.com/blackrock/axis/common"
	"github.com/blackrock/axis/pkg/apis/sensor/v1alpha1"
	"github.com/blackrock/axis/store"
)

// check all the signal statuses and if they are all resolved and constraints are met, let's create the trigger event
func (soc *sOperationCtx) processTrigger(trigger v1alpha1.Trigger) (*v1alpha1.NodeStatus, error) {
	soc.log.Debugf("evaluating trigger '%s'", trigger.Name)
	node := soc.getNodeByName(trigger.Name)
	if node != nil && node.IsComplete() {
		return node, nil
	}

	if node == nil {
		node = soc.initializeNode(trigger.Name, v1alpha1.NodeTypeTrigger, v1alpha1.NodePhaseActive)
	}

	if node.Phase != v1alpha1.NodePhaseResolved && node.Phase != v1alpha1.NodePhaseSucceeded {
		err := soc.executeTrigger(trigger)
		if err != nil {
			return soc.markNodePhase(trigger.Name, v1alpha1.NodePhaseError, err.Error()), err
		}
	}
	return soc.markNodePhase(trigger.Name, v1alpha1.NodePhaseSucceeded), nil
}

// execute the trigger
func (soc *sOperationCtx) executeTrigger(trigger v1alpha1.Trigger) error {
	if trigger.Message != nil {
		err := sendMessage(trigger.Message)
		if err != nil {
			soc.log.Warn("failed to send message. cause: %s", err.Error())
			return err
		}
	}
	if trigger.Resource != nil {
		creds, err := store.GetCredentials(soc.controller.kubeClientset, soc.controller.Config.Namespace, trigger.Resource.ArtifactLocation)
		if err != nil {
			return err
		}
		reader, err := store.GetArtifactReader(trigger.Resource.ArtifactLocation, creds)
		if err != nil {
			return err
		}
		uObj, err := store.FetchArtifact(reader, trigger.Resource.GroupVersionKind)
		if err != nil {
			return err
		}
		err = soc.createResourceObject(trigger.Resource, uObj)
		if err != nil {
			return err
		}
	}
	return nil
}

func sendMessage(message *v1alpha1.Message) error {
	payload := []byte(message.Body)
	switch message.Stream.GetType() {
	case v1alpha1.StreamTypeNats:
		natsConnection, err := nats.Connect(message.Stream.NATS.URL)
		if err != nil {
			return err
		}
		defer natsConnection.Close()
		return natsConnection.Publish(message.Stream.NATS.Subject, payload)
	default:
		return fmt.Errorf("unsupported type of stream %s", message.Stream.GetType())
	}
}

func (soc *sOperationCtx) createResourceObject(resource *v1alpha1.ResourceObject, obj *unstructured.Unstructured) error {
	if resource.Namespace != "" {
		obj.SetNamespace(resource.Namespace)
	}
	if resource.Labels != nil {
		labels := obj.GetLabels()
		if labels != nil {
			for k, v := range resource.Labels {
				//TODO: check if override?
				labels[k] = v
			}
			obj.SetLabels(labels)
		}
		obj.SetLabels(resource.Labels)
	}

	gvk := obj.GroupVersionKind()
	clientPool := dynamic.NewDynamicClientPool(soc.controller.kubeConfig)
	disco, err := discovery.NewDiscoveryClientForConfig(soc.controller.kubeConfig)
	if err != nil {
		return err
	}
	client, err := clientPool.ClientForGroupVersionKind(gvk)
	if err != nil {
		return err
	}

	apiResource, err := common.ServerResourceForGroupVersionKind(disco, gvk)
	if err != nil {
		return err
	}
	soc.log.Debugf("chose api '%s' for %s", apiResource.Name, gvk)

	reIf := client.Resource(apiResource, soc.controller.Config.Namespace)
	liveObj, err := reIf.Create(obj)
	if err == nil {
		soc.log.Infof("%s '%s' created", liveObj.GetKind(), liveObj.GetName())
		return nil
	}
	if !errors.IsAlreadyExists(err) {
		return err
	}
	liveObj, err = reIf.Get(obj.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	//todo: implement a diff between obj and liveObj
	soc.log.Info("%s '%s' already exists", liveObj.GetKind(), liveObj.GetName())
	return nil
}
