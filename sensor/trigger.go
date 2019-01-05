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

package sensor

import (
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/controllers/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	v1alpha "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/store"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)


// processTriggers checks if all event dependencies are complete and then starts executing triggers
func (sec *sensorExecutionCtx) processTriggers() {
	// to trigger the sensor action/s we need to check if all event dependencies are completed and sensor is active
	if sec.sensor.AreAllNodesSuccess(v1alpha1.NodeTypeEventDependency) {
		sec.log.Info().Msg("all event dependencies are marked completed")
		for _, trigger := range sec.sensor.Spec.Triggers {
			err := sec.executeTrigger(trigger)
			if err != nil {
				sec.log.Error().Str("trigger-name", trigger.Name).Err(err).Msg("trigger failed to execute")
				// todo: escalate using K8s event
				continue
			}
			// mark trigger as complete.
			// todo: generate K8 event marking trigger as successful
		}
		return
	}
	sec.log.Info().Msg("triggers can't be executed because either event dependencies are not complete")
}

// execute the trigger
func (sec *sensorExecutionCtx) executeTrigger(trigger v1alpha1.Trigger) error {
	if trigger.Resource != nil {
		creds, err := store.GetCredentials(sec.kubeClient, sec.sensor.Namespace, &trigger.Resource.Source)
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
		err = sec.createResourceObject(trigger.Resource, uObj)
		if err != nil {
			return err
		}
	}
	return nil
}

// createResourceObject creates K8s object for trigger
func (sec *sensorExecutionCtx) createResourceObject(resource *v1alpha1.ResourceObject, obj *unstructured.Unstructured) error {
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
	// 2. extract the appropriate eventDependency events based on the resource params
	// 3. apply the params to the JSON object
	// 4. unmarshal the obj from the updated JSON
	if len(resource.Parameters) > 0 {
		jObj, err := obj.MarshalJSON()
		if err != nil {
			return err
		}
		events := sec.extractSignalEvents(resource.Parameters)
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
	client, err := sec.clientPool.ClientForGroupVersionKind(gvk)
	if err != nil {
		return err
	}

	apiResource, err := common.ServerResourceForGroupVersionKind(sec.discoveryClient, gvk)
	if err != nil {
		return err
	}
	sec.log.Info().Str("api", apiResource.Name).Str("group-version", gvk.Version).Msg("created api resource")

	reIf := client.Resource(apiResource, resource.Namespace)
	liveObj, err := reIf.Create(obj)
	if err != nil {
		return err
	}
	sec.log.Info().Str("kind", liveObj.GetKind()).Str("name", liveObj.GetName()).Msg("created object")
	if !errors.IsAlreadyExists(err) {
		return err
	}
	liveObj, err = reIf.Get(obj.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	//todo: implement a diff between obj and liveObj
	sec.log.Warn().Str("kind", liveObj.GetKind()).Str("name", liveObj.GetName()).Msg("object already exist")
	return nil
}

// helper method to extract the events from the signals associated with the resource params
// returns a map of the events keyed by the eventDependency name
func (sec *sensorExecutionCtx) extractSignalEvents(params []v1alpha1.ResourceParameter) map[string]v1alpha.Event {
	events := make(map[string]v1alpha.Event)
	for _, param := range params {
		if param.Src != nil {
			node := sensor.GetNodeByName(sec.sensor, param.Src.Signal)
			if node == nil {
				sec.log.Warn().Str("param-src", param.Src.Signal).Str("param-dest", param.Dest).Msg("WARNING: eventDependency node does not exist, cannot apply parameter")
				continue
			}
			if node.Event == nil {
				sec.log.Warn().Str("param-src", param.Src.Signal).Str("param-dest", param.Dest).Msg("WARNING: eventDependency node does not exist, cannot apply parameter")
				continue
			}
			events[param.Src.Signal] = node.Event.Event
		}
	}
	return events
}
