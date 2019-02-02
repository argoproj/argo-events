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

package sensors

import (
	"fmt"
	"github.com/Knetic/govaluate"

	"github.com/argoproj/argo-events/common"
	sn "github.com/argoproj/argo-events/controllers/sensor"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/store"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// canProcessTriggers evaluates whether triggers can be processed and executed
func (sec *sensorExecutionCtx) canProcessTriggers() (bool, error) {
	if sec.sensor.Spec.DependencyGroups != nil {
		groups := make(map[string]interface{}, len(sec.sensor.Spec.DependencyGroups))
	group:
		for _, group := range sec.sensor.Spec.DependencyGroups {
			for _, dependency := range group.Dependencies {
				if nodeStatus := sn.GetNodeByName(sec.sensor, dependency); nodeStatus.Phase != v1alpha1.NodePhaseComplete {
					continue group
				}
			}
			groups[group.Name] = true
		}

		expression, err := govaluate.NewEvaluableExpression(sec.sensor.Spec.Circuit)
		if err != nil {
			return false, fmt.Errorf("failed to create circuit expression. err: %+v", err)
		}

		result, err := expression.Evaluate(groups)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate circuit expression. err: %+v", err)
		}

		return result == true, err
	}

	if sec.sensor.AreAllNodesSuccess(v1alpha1.NodeTypeEventDependency) {
		return true, nil
	}

	return false, nil
}

// processTriggers checks if all event dependencies are complete and then starts executing triggers
func (sec *sensorExecutionCtx) processTriggers() {
	// to trigger the sensor action/s we need to check if all event dependencies are completed and sensor is active
	sec.log.Info().Msg("all event dependencies are marked completed, processing triggers")

	// labels for K8s event
	labels := map[string]string{
		common.LabelSensorName: sec.sensor.Name,
		common.LabelOperation:  "process_triggers",
	}

	for _, trigger := range sec.sensor.Spec.Triggers {
		if err := sec.executeTrigger(trigger); err != nil {
			sec.log.Error().Str("trigger-name", trigger.Name).Err(err).Msg("trigger failed to execute")

			sn.MarkNodePhase(sec.sensor, trigger.Name, v1alpha1.NodeTypeTrigger, v1alpha1.NodePhaseError, nil, &sec.log, fmt.Sprintf("failed to execute trigger. err: %+v", err))

			// escalate using K8s event
			labels[common.LabelEventType] = string(common.EscalationEventType)
			if err := common.GenerateK8sEvent(sec.kubeClient, fmt.Sprintf("failed to execute trigger %s", trigger.Name), common.EscalationEventType,
				"trigger failure", sec.sensor.Name, sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
				sec.log.Error().Err(err).Msg("failed to create K8s event to escalate trigger failure")
			}
			continue
		}

		// mark trigger as complete.
		sn.MarkNodePhase(sec.sensor, trigger.Name, v1alpha1.NodeTypeTrigger, v1alpha1.NodePhaseComplete, nil, &sec.log, "successfully executed trigger")

		labels[common.LabelEventType] = string(common.OperationSuccessEventType)
		if err := common.GenerateK8sEvent(sec.kubeClient, fmt.Sprintf("trigger %s executed successfully", trigger.Name), common.OperationSuccessEventType,
			"trigger executed", sec.sensor.Name, sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
			sec.log.Error().Err(err).Msg("failed to create K8s event to log trigger execution")
		}
	}

	// increment completion counter
	sec.sensor.Status.CompletionCount = sec.sensor.Status.CompletionCount + 1

	// create K8s event to mark the trigger round completion
	labels[common.LabelEventType] = string(common.OperationSuccessEventType)
	if err := common.GenerateK8sEvent(sec.kubeClient, fmt.Sprintf("completion count:%d", sec.sensor.Status.CompletionCount), common.OperationSuccessEventType,
		"triggers execution round completion", sec.sensor.Name, sec.sensor.Namespace, sec.controllerInstanceID, sensor.Kind, labels); err != nil {
		sec.log.Error().Err(err).Msg("failed to create K8s event to log trigger execution round completion")
	}

	// Mark all signal nodes as active
	for _, dependency := range sec.sensor.Spec.Dependencies {
		sn.MarkNodePhase(sec.sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, &sec.log, "dependency is re-activated")
	}
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
		if err = sec.createResourceObject(trigger.Resource, uObj); err != nil {
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
			return fmt.Errorf("failed to marshal json. err: %+v", err)
		}
		events := sec.extractEvents(resource.Parameters)
		jUpdatedObj, err := applyParams(jObj, resource.Parameters, events)
		if err != nil {
			return fmt.Errorf("failed to apply params. err: %+v", err)
		}
		err = obj.UnmarshalJSON(jUpdatedObj)
		if err != nil {
			return fmt.Errorf("failed to un-marshal json. err: %+v", err)
		}
	}

	gvk := obj.GroupVersionKind()
	client, err := sec.clientPool.ClientForGroupVersionKind(gvk)
	if err != nil {
		return fmt.Errorf("failed to get client for given group verison and kind. err: %+v", err)
	}

	apiResource, err := common.ServerResourceForGroupVersionKind(sec.discoveryClient, gvk)
	if err != nil {
		return fmt.Errorf("failed to get server resource for given group verison and kind. err: %+v", err)
	}
	sec.log.Info().Str("api", apiResource.Name).Str("group-version", gvk.Version).Msg("created api resource")

	reIf := client.Resource(apiResource, resource.Namespace)
	liveObj, err := reIf.Create(obj)
	if err != nil {
		return fmt.Errorf("failed to create resource object. err: %+v", err)
	}
	sec.log.Info().Str("kind", liveObj.GetKind()).Str("name", liveObj.GetName()).Msg("created object")
	if !errors.IsAlreadyExists(err) {
		return err
	}
	liveObj, err = reIf.Get(obj.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	sec.log.Warn().Str("kind", liveObj.GetKind()).Str("name", liveObj.GetName()).Msg("object already exist")
	return nil
}

// helper method to extract the events from the event dependencies nodes associated with the resource params
// returns a map of the events keyed by the event dependency name
func (sec *sensorExecutionCtx) extractEvents(params []v1alpha1.ResourceParameter) map[string]apicommon.Event {
	events := make(map[string]apicommon.Event)
	for _, param := range params {
		if param.Src != nil {
			node := sn.GetNodeByName(sec.sensor, param.Src.Event)
			if node == nil {
				sec.log.Warn().Str("param-src", param.Src.Event).Str("param-dest", param.Dest).Msg("WARNING: event dependency node does not exist, cannot apply parameter")
				continue
			}
			if node.Event == nil {
				sec.log.Warn().Str("param-src", param.Src.Event).Str("param-dest", param.Dest).Msg("WARNING: event dependency node does not exist, cannot apply parameter")
				continue
			}
			events[param.Src.Event] = *node.Event
		}
	}
	return events
}
