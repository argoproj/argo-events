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
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Knetic/govaluate"
	"github.com/argoproj/argo-events/common"
	sn "github.com/argoproj/argo-events/controllers/sensor"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/store"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
)

// canProcessTriggers evaluates whether triggers can be processed and executed
func (sensorCtx *sensorContext) canProcessTriggers() (bool, error) {
	if sensorCtx.sensor.Spec.ErrorOnFailedRound && sensorCtx.sensor.Status.TriggerCycleStatus == v1alpha1.TriggerCycleFailure {
		return false, fmt.Errorf("last trigger cycle was a failure and sensor policy is set to ErrorOnFailedRound, so won't process the triggers")
	}

	if sensorCtx.sensor.Spec.DependencyGroups != nil {
		groups := make(map[string]interface{}, len(sensorCtx.sensor.Spec.DependencyGroups))
	group:
		for _, group := range sensorCtx.sensor.Spec.DependencyGroups {
			for _, dependency := range group.Dependencies {
				if nodeStatus := sn.GetNodeByName(sensorCtx.sensor, dependency); nodeStatus.Phase != v1alpha1.NodePhaseComplete {
					groups[group.Name] = false
					continue group
				}
			}
			sn.MarkNodePhase(sensorCtx.sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseComplete, nil, sensorCtx.logger, "dependency group is complete")
			groups[group.Name] = true
		}

		expression, err := govaluate.NewEvaluableExpression(sensorCtx.sensor.Spec.Circuit)
		if err != nil {
			return false, fmt.Errorf("failed to create circuit expression. err: %+v", err)
		}

		result, err := expression.Evaluate(groups)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate circuit expression. err: %+v", err)
		}

		return result == true, err
	}

	if sensorCtx.sensor.AreAllNodesSuccess(v1alpha1.NodeTypeEventDependency) {
		return true, nil
	}

	return false, nil
}

// canExecuteTrigger determines whether a trigger is executable based on condition set on trigger
func (sensorCtx *sensorContext) canExecuteTrigger(trigger v1alpha1.Trigger) bool {
	if trigger.Template.When == nil {
		return true
	}
	if trigger.Template.When.Any != nil {
		for _, group := range trigger.Template.When.Any {
			if status := sn.GetNodeByName(sensorCtx.sensor, group); status.Type == v1alpha1.NodeTypeDependencyGroup && status.Phase == v1alpha1.NodePhaseComplete {
				return true
			}
		}
		return false
	}
	if trigger.Template.When.All != nil {
		for _, group := range trigger.Template.When.All {
			if status := sn.GetNodeByName(sensorCtx.sensor, group); status.Type == v1alpha1.NodeTypeDependencyGroup && status.Phase != v1alpha1.NodePhaseComplete {
				return false
			}
		}
		return true
	}
	return true
}

// processTriggers checks if all event dependencies are complete and then starts executing triggers
func (sensorCtx *sensorContext) processTriggers() {
	// to trigger the sensor action/s we need to check if all event dependencies are completed and sensor is active
	sensorCtx.logger.Info("all event dependencies are marked completed, processing triggers")

	// labels for K8s event
	labels := map[string]string{
		common.LabelSensorName: sensorCtx.sensor.Name,
		common.LabelOperation:  "process_triggers",
	}

	successTriggerCycle := true

	for _, trigger := range sensorCtx.sensor.Spec.Triggers {
		log := sensorCtx.logger.WithField(common.LabelTriggerName, trigger.Template.Name)

		// check if a trigger condition is set
		if canExecute := sensorCtx.canExecuteTrigger(trigger); !canExecute {
			log.Info("trigger can't be executed because trigger condition failed")
			continue
		}

		if err := sensorCtx.executeTrigger(trigger); err != nil {
			log.WithError(err).Error("trigger failed to execute")

			// mark trigger status as error
			sn.MarkNodePhase(sensorCtx.sensor, trigger.Template.Name, v1alpha1.NodeTypeTrigger, v1alpha1.NodePhaseError, nil, sensorCtx.logger, fmt.Sprintf("failed to execute trigger.Template. err: %+v", err))

			successTriggerCycle = false

			// escalate using K8s event
			labels[common.LabelEventType] = string(common.EscalationEventType)
			if err := common.GenerateK8sEvent(sensorCtx.kubeClient, fmt.Sprintf("failed to execute trigger %s", trigger.Template.Name), common.EscalationEventType,
				"trigger failure", sensorCtx.sensor.Name, sensorCtx.sensor.Namespace, sensorCtx.controllerInstanceID, sensor.Kind, labels); err != nil {
				log.WithError(err).Error("failed to create K8s event to escalate trigger failure")
			}
			continue
		}

		// mark trigger as complete.
		sn.MarkNodePhase(sensorCtx.sensor, trigger.Template.Name, v1alpha1.NodeTypeTrigger, v1alpha1.NodePhaseComplete, nil, sensorCtx.logger, "successfully executed trigger")

		labels[common.LabelEventType] = string(common.OperationSuccessEventType)
		if err := common.GenerateK8sEvent(sensorCtx.kubeClient, fmt.Sprintf("trigger %s executed successfully", trigger.Template.Name), common.OperationSuccessEventType,
			"trigger executed", sensorCtx.sensor.Name, sensorCtx.sensor.Namespace, sensorCtx.controllerInstanceID, sensor.Kind, labels); err != nil {
			log.WithError(err).Error("failed to create K8s event to logger trigger execution")
		}
	}

	// increment completion counter
	sensorCtx.sensor.Status.TriggerCycleCount = sensorCtx.sensor.Status.TriggerCycleCount + 1
	// set completion time
	sensorCtx.sensor.Status.LastCycleTime = metav1.Now()

	if successTriggerCycle {
		sensorCtx.sensor.Status.TriggerCycleStatus = v1alpha1.TriggerCycleSuccess
	} else {
		sensorCtx.sensor.Status.TriggerCycleStatus = v1alpha1.TriggerCycleFailure
	}

	// create K8s event to mark the trigger round completion
	labels[common.LabelEventType] = string(common.OperationSuccessEventType)
	if err := common.GenerateK8sEvent(sensorCtx.kubeClient, fmt.Sprintf("completion count:%d", sensorCtx.sensor.Status.TriggerCycleCount), common.OperationSuccessEventType,
		"triggers execution round completion", sensorCtx.sensor.Name, sensorCtx.sensor.Namespace, sensorCtx.controllerInstanceID, sensor.Kind, labels); err != nil {
		sensorCtx.logger.WithError(err).Error("failed to create K8s event to logger trigger execution round completion")
	}

	// Mark all dependency nodes as active
	for _, dependency := range sensorCtx.sensor.Spec.Dependencies {
		sn.MarkNodePhase(sensorCtx.sensor, dependency.Name, v1alpha1.NodeTypeEventDependency, v1alpha1.NodePhaseActive, nil, sensorCtx.logger, "dependency is re-activated")
	}
	// Mark all dependency groups as active
	for _, group := range sensorCtx.sensor.Spec.DependencyGroups {
		sn.MarkNodePhase(sensorCtx.sensor, group.Name, v1alpha1.NodeTypeDependencyGroup, v1alpha1.NodePhaseActive, nil, sensorCtx.logger, "dependency group is re-activated")
	}
}

// applyParamsTrigger applies parameters to trigger
func (sensorCtx *sensorContext) applyParamsTrigger(trigger *v1alpha1.Trigger) error {
	if trigger.TemplateParameters != nil && len(trigger.TemplateParameters) > 0 {
		templateBytes, err := json.Marshal(trigger.Template)
		if err != nil {
			return err
		}
		tObj, err := applyParams(templateBytes, trigger.TemplateParameters, sensorCtx.extractEvents(trigger.TemplateParameters))
		if err != nil {
			return err
		}
		template := &v1alpha1.TriggerTemplate{}
		if err = json.Unmarshal(tObj, template); err != nil {
			return err
		}
		trigger.Template = template
	}
	return nil
}

// applyParamsResource applies parameters to resource within trigger
func (sensorCtx *sensorContext) applyParamsResource(parameters []v1alpha1.TriggerParameter, obj *unstructured.Unstructured) error {
	if parameters != nil && len(parameters) > 0 {
		jObj, err := obj.MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal json. err: %+v", err)
		}
		events := sensorCtx.extractEvents(parameters)
		jUpdatedObj, err := applyParams(jObj, parameters, events)
		if err != nil {
			return fmt.Errorf("failed to apply params. err: %+v", err)
		}
		err = obj.UnmarshalJSON(jUpdatedObj)
		if err != nil {
			return fmt.Errorf("failed to un-marshal json. err: %+v", err)
		}
	}
	return nil
}

// executeTrigger executes the trigger
func (sensorCtx *sensorContext) executeTrigger(trigger v1alpha1.Trigger) error {
	if trigger.Template != nil {
		if err := sensorCtx.applyParamsTrigger(&trigger); err != nil {
			return err
		}
		creds, err := store.GetCredentials(sensorCtx.kubeClient, sensorCtx.sensor.Namespace, trigger.Template.Source)
		if err != nil {
			return err
		}
		reader, err := store.GetArtifactReader(trigger.Template.Source, creds, sensorCtx.kubeClient)
		if err != nil {
			return err
		}
		uObj, err := store.FetchArtifact(reader, trigger.Template.GroupVersionResource)
		if err != nil {
			return err
		}
		if err = sensorCtx.createResourceObject(&trigger, uObj); err != nil {
			return err
		}
	}
	return nil
}

// applyTriggerPolicy applies backoff and evaluates success/failure for a trigger
func (sensorCtx *sensorContext) applyTriggerPolicy(trigger *v1alpha1.Trigger, resourceInterface dynamic.NamespaceableResourceInterface, name, namespace string) error {
	err := wait.ExponentialBackoff(wait.Backoff{
		Duration: trigger.Policy.Backoff.Duration,
		Steps:    trigger.Policy.Backoff.Steps,
		Factor:   trigger.Policy.Backoff.Factor,
		Jitter:   trigger.Policy.Backoff.Jitter,
	}, func() (bool, error) {
		obj, err := resourceInterface.Namespace(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			sensorCtx.logger.WithError(err).WithField("resource-name", obj.GetName()).Error("failed to get triggered resource")
			return false, nil
		}

		labels := obj.GetLabels()
		if labels == nil {
			sensorCtx.logger.Warn("triggered object does not have labels, won't apply the trigger policy")
			return false, nil
		}
		sensorCtx.logger.WithField("labels", labels).Debug("object labels")

		// check if success labels match with labels on object
		if trigger.Policy.State.Success != nil {
			success := true
			for successKey, successValue := range trigger.Policy.State.Success {
				if value, ok := labels[successKey]; ok {
					if successValue != value {
						success = false
						break
					}
					continue
				}
				success = false
			}
			if success {
				return true, nil
			}
		}

		// check if failure labels match with labels on object
		if trigger.Policy.State.Failure != nil {
			failure := true
			for failureKey, failureValue := range trigger.Policy.State.Failure {
				if value, ok := labels[failureKey]; ok {
					if failureValue != value {
						failure = false
						break
					}
					continue
				}
				failure = false
			}
			if failure {
				return false, fmt.Errorf("trigger is in failed state")
			}
		}

		return false, nil
	})

	return err
}

// createResourceObject creates K8s object for trigger
func (sensorCtx *sensorContext) createResourceObject(trigger *v1alpha1.Trigger, obj *unstructured.Unstructured) error {
	// passing parameters to the resource object requires 4 steps
	// 1. marshaling the obj to JSON
	// 2. extract the appropriate eventDependency events based on the resource params
	// 3. apply the params to the JSON object
	// 4. unmarshal the obj from the updated JSON
	if err := sensorCtx.applyParamsResource(trigger.ResourceParameters, obj); err != nil {
		return err
	}

	namespace := obj.GetNamespace()
	// Defaults to sensor's namespace
	if namespace == "" {
		namespace = sensorCtx.sensor.Namespace
	}
	obj.SetNamespace(namespace)

	dynamicResInterface := sensorCtx.dynamicClient.Resource(schema.GroupVersionResource{
		Group:    trigger.Template.Group,
		Version:  trigger.Template.Version,
		Resource: trigger.Template.Resource,
	})

	liveObj, err := dynamicResInterface.Namespace(namespace).Create(obj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create resource object. err: %+v", err)
	}

	sensorCtx.logger.WithFields(
		map[string]interface{}{
			"kind": liveObj.GetKind(),
			"name": liveObj.GetName(),
		}).Info("created object")

	// apply trigger policy if set
	if trigger.Policy != nil {
		sensorCtx.logger.Info("applying trigger policy...")

		if err := sensorCtx.applyTriggerPolicy(trigger, dynamicResInterface, liveObj.GetName(), namespace); err != nil {
			if err == wait.ErrWaitTimeout {
				if trigger.Policy.ErrorOnBackoffTimeout {
					return fmt.Errorf("failed to determine status of the triggered resource. setting trigger state as failed")
				}
				return nil
			}
			return err
		}
	}

	return nil
}

// helper method to extract the events from the event dependencies nodes associated with the resource params
// returns a map of the events keyed by the event dependency name
func (sensorCtx *sensorContext) extractEvents(params []v1alpha1.TriggerParameter) map[string]apicommon.Event {
	events := make(map[string]apicommon.Event)
	for _, param := range params {
		if param.Src != nil {
			log := sensorCtx.logger.WithFields(
				map[string]interface{}{
					"param-src":  param.Src.Event,
					"param-dest": param.Dest,
				})
			node := sn.GetNodeByName(sensorCtx.sensor, param.Src.Event)
			if node == nil {
				log.Warn("event dependency node does not exist, cannot apply parameter")
				continue
			}
			if node.Event == nil {
				log.Warn("event in event dependency does not exist, cannot apply parameter")
				continue
			}
			events[param.Src.Event] = *node.Event
		}
	}
	return events
}
