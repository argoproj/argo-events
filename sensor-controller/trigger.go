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

package sensor_controller

import (
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	v1alpha "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/store"
)

// execute the trigger
func (sc *sensorExecutor) executeTrigger(trigger v1alpha1.Trigger) error {
	if trigger.Resource != nil {
		creds, err := store.GetCredentials(sc.kubeClient, sc.sensor.Namespace, &trigger.Resource.Source)
		if err != nil {
			return err
		}
		reader, err := store.GetArtifactReader(&trigger.Resource.Source, creds)
		if err != nil {
			return err
		}
		uObj, err := store.FetchArtifact(reader, trigger.Resource.GroupVersionKind)
		if err != nil {
			return err
		}
		err = sc.createResourceObject(trigger.Resource, uObj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sc *sensorExecutor) createResourceObject(resource *v1alpha1.ResourceObject, obj *unstructured.Unstructured) error {
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

	// passing parameters to the resource object requires 4 steps
	// 1. marshaling the obj to JSON
	// 2. extract the appropriate signal events based on the resource params
	// 3. apply the params to the JSON object
	// 4. unmarshal the obj from the updated JSON
	if len(resource.Parameters) > 0 {
		jObj, err := obj.MarshalJSON()
		if err != nil {
			return err
		}
		events := sc.extractSignalEvents(resource.Parameters)
		jUpdatedObj, err := applyParams(jObj, resource.Parameters, events)
		if err != nil {
			return err
		}
		err = obj.UnmarshalJSON(jUpdatedObj)
		if err != nil {
			return err
		}
	}

	gvk := obj.GroupVersionKind()
	clientPool := dynamic.NewDynamicClientPool(sc.config)
	disco, err := discovery.NewDiscoveryClientForConfig(sc.config)
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
	sc.log.Debug().Str("api", apiResource.Name).Str("group-version", gvk.Version).Msg("created api resource")

	reIf := client.Resource(apiResource, sc.sensor.Namespace)
	liveObj, err := reIf.Create(obj)
	if err != nil {
		return err
	}
	sc.log.Info().Str("kind", liveObj.GetKind()).Str("name", liveObj.GetName()).Msg("created object")
	if !errors.IsAlreadyExists(err) {
		return err
	}
	liveObj, err = reIf.Get(obj.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	//todo: implement a diff between obj and liveObj
	sc.log.Warn().Str("kind", liveObj.GetKind()).Str("name", liveObj.GetName()).Msg("object already exist")
	return nil
}

// helper method to extract the events from the signals associated with the resource params
// returns a map of the events keyed by the signal name
func (sc *sensorExecutor) extractSignalEvents(params []v1alpha1.ResourceParameter) map[string]v1alpha.Event {
	events := make(map[string]v1alpha.Event)
	for _, param := range params {
		if param.Src != nil {
			node := getNodeByName(sc.sensor, param.Src.Signal)
			if node == nil {
				sc.log.Warn().Str("param-src", param.Src.Signal).Str("param-dest", param.Dest).Msg("WARNING: signal node does not exist, cannot apply parameter")
				continue
			}
			if node.LatestEvent == nil {
				sc.log.Warn().Str("param-src", param.Src.Signal).Str("param-dest", param.Dest).Msg("WARNING: signal node does not exist, cannot apply parameter")
				continue
			}
			events[param.Src.Signal] = node.LatestEvent.Event
		}
	}
	return events
}
