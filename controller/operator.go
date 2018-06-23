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
	"runtime/debug"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	client "github.com/argoproj/argo-events/pkg/client/clientset/versioned/typed/sensor/v1alpha1"
)

// the context of an operation on a sensor.
// the controller creates this context each time it picks a Sensor off its queue.
type sOperationCtx struct {
	// s is the sensor object
	s *v1alpha1.Sensor

	// events contains the latest events for each signal of this sensor
	// this gets populated during operating
	// allows passing of events from signals -> triggers
	events map[string]v1alpha1.Event

	// updated indicates whether the sensor object was updated and needs to be persisted back to k8
	updated bool

	// log is the log of this context
	log *zap.SugaredLogger

	// reference to the sensor controller
	controller *SensorController
}

// newSensorOperationCtx creates and initializes a new sOperationCtx object
func newSensorOperationCtx(s *v1alpha1.Sensor, controller *SensorController) *sOperationCtx {
	return &sOperationCtx{
		s:          s.DeepCopy(),
		events:     make(map[string]v1alpha1.Event),
		updated:    false,
		log:        controller.log.With(zap.String("sensor", s.Name), zap.String("namespace", s.Namespace)),
		controller: controller,
	}
}

func (soc *sOperationCtx) operate() error {
	defer soc.persistUpdates()
	defer func() {
		if r := recover(); r != nil {
			if rerr, ok := r.(error); ok {
				soc.markSensorPhase(v1alpha1.NodePhaseError, true, rerr.Error())
				soc.log.Error(rerr)
			}
			soc.log.Errorf("recovered from panic: %+v\n%s", r, debug.Stack())
		}
	}()

	if soc.s.Status.Phase == v1alpha1.NodePhaseNew {
		// perform one-time sensor validation
		err := validateSensor(soc.s)
		if err != nil {
			soc.markSensorPhase(v1alpha1.NodePhaseError, true, err.Error())
			return fmt.Errorf("validation failed due to %s", err.Error())
		}
	}

	// process the sensor's signals
	for _, signal := range soc.s.Spec.Signals {
		_, err := soc.processSignal(signal)
		if err != nil {
			soc.markNodePhase(signal.Name, v1alpha1.NodePhaseError, err.Error())
			return err
		}
	}

	// process the triggers if all sensor signals are resolved/successful
	// this means we can start processing triggers when signals are resolved, not completed so may introduce discrepancy if signal fails to complete after being resolved
	if soc.s.AreAllNodesSuccess(v1alpha1.NodeTypeSignal) {
		for _, trigger := range soc.s.Spec.Triggers {
			_, err := soc.processTrigger(trigger)
			if err != nil {
				soc.log.Errorf("trigger %s failed to execute. %s", trigger.Name, err.Error())
				soc.markNodePhase(trigger.Name, v1alpha1.NodePhaseError, err.Error())
				soc.markSensorPhase(v1alpha1.NodePhaseError, false, err.Error())
				return err
			}
		}

		if soc.s.AreAllNodesSuccess(v1alpha1.NodeTypeTrigger) {
			// here we need to check if the sensor is repeatable, if so, we should go back to init phase for the sensor & all the nodes
			// todo: add spec level deadlines here
			if soc.s.Spec.Repeat {
				soc.reRunSensor()
			} else {
				soc.markSensorPhase(v1alpha1.NodePhaseComplete, true)
			}
			return nil
		}
	}

	if soc.s.IsComplete() {
		// escalate
		if !soc.s.Status.Escalated {
			soc.log.Warnf("escalating sensor to level %s via %s message", soc.s.Spec.Escalation.Level, soc.s.Spec.Escalation.Message.Stream.Type)
			err := sendMessage(&soc.s.Spec.Escalation.Message)
			if err != nil {
				return err
			}
			soc.s.Status.Escalated = true
			soc.updated = true
		} else {
			soc.log.Debug("sensor already escalated")
		}
	}

	return nil
}

func (soc *sOperationCtx) reRunSensor() {
	// if we get here we know the sensor pod & job is succeeded, the triggers have fired, but the sensor is repeatable
	// we know have to reset the sensor status
	// todo: persist changes in a transaction store somewhere, is it reasonable to put in the sensor object? probably not for scale

	soc.log.Info("resetting nodes and re-running sensor")
	// reset the nodes
	soc.s.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
	// re-initialize the signal nodes
	for _, signal := range soc.s.Spec.Signals {
		soc.initializeNode(signal.Name, v1alpha1.NodeTypeSignal, v1alpha1.NodePhaseNew)
	}

	// add a field to status to log # of times this sensor went resolved -> init
	soc.markSensorPhase(v1alpha1.NodePhaseNew, false)
	soc.updated = true
}

// persist the updates to the Sensor resource
func (soc *sOperationCtx) persistUpdates() {
	var err error
	if !soc.updated {
		return
	}
	sensorClient := soc.controller.sensorClientset.ArgoprojV1alpha1().Sensors(soc.s.ObjectMeta.Namespace)
	soc.s, err = sensorClient.Update(soc.s)
	if err != nil {
		soc.log.Warnf("error updating sensor: %v", err)
		if errors.IsConflict(err) {
			return
		}
		soc.log.Info("re-applying updates on latest version and retrying update")
		err = soc.reapplyUpdate(sensorClient)
		if err != nil {
			soc.log.Infof("failed to re-apply update: %+v", err)
			return
		}
	}
	soc.log.Info("sensor update successful")

	time.Sleep(1 * time.Second)
}

// todo: implement me
func (soc *sOperationCtx) reapplyUpdate(sensorClient client.SensorInterface) error {
	attempt := 1
	for {
		_, err := sensorClient.Get(soc.s.ObjectMeta.Name, metav1.GetOptions{})

		if attempt > 5 {
			return err
		}
	}
}

// create a new node
func (soc *sOperationCtx) initializeNode(nodeName string, nodeType v1alpha1.NodeType, phase v1alpha1.NodePhase, messages ...string) *v1alpha1.NodeStatus {
	if soc.s.Status.Nodes == nil {
		soc.s.Status.Nodes = make(map[string]v1alpha1.NodeStatus)
	}
	nodeID := soc.s.NodeID(nodeName)
	oldNode, ok := soc.s.Status.Nodes[nodeID]
	if ok {
		soc.log.Infof("node %s already initialized", nodeName)
		return &oldNode
	}
	node := v1alpha1.NodeStatus{
		ID:          nodeID,
		Name:        nodeName,
		DisplayName: nodeName,
		Type:        nodeType,
		Phase:       phase,
		StartedAt:   metav1.Time{Time: time.Now().UTC()},
	}
	if len(messages) > 0 {
		node.Message = messages[0]
	}
	soc.s.Status.Nodes[nodeID] = node
	soc.log.Infof("%s '%s' initialized", node.Type, node.DisplayName)
	soc.updated = true
	return &node
}

// mark the node with a phase, retuns the node
func (soc *sOperationCtx) markNodePhase(nodeName string, phase v1alpha1.NodePhase, message ...string) *v1alpha1.NodeStatus {
	node := soc.getNodeByName(nodeName)
	if node == nil {
		soc.log.Panicf("node '%s' uninitialized", nodeName)
	}
	if node.Phase != phase {
		soc.log.Infof("%s '%s' phase %s -> %s", node.Type, nodeName, node.Phase, phase)
		node.Phase = phase
		soc.updated = true
	}
	if len(message) > 0 && node.Message != message[0] {
		soc.log.Infof("%s '%s' message %s -> %s", node.Type, node.Name, node.Message, message[0])
		node.Message = message[0]
		soc.updated = true
	}
	if node.IsComplete() && node.CompletedAt.IsZero() {
		node.CompletedAt = metav1.Time{Time: time.Now().UTC()}
		soc.log.Infof("%s '%s' completed: %s", node.Type, node.Name, node.CompletedAt)
		soc.updated = true
	}
	soc.s.Status.Nodes[node.ID] = *node
	return node
}

// mark the overall sensor phase
func (soc *sOperationCtx) markSensorPhase(phase v1alpha1.NodePhase, markResolved bool, message ...string) {
	phaseChange := soc.s.Status.Phase != phase
	if phaseChange {
		soc.log.Infof("sensor phase %s -> %s", soc.s.Status.Phase, phase)
		soc.s.Status.Phase = phase
		soc.updated = true
		if soc.s.ObjectMeta.Labels == nil {
			soc.s.ObjectMeta.Labels = make(map[string]string)
		}
		soc.s.ObjectMeta.Labels[common.LabelKeyPhase] = string(phase)
	}
	if soc.s.Status.StartedAt.IsZero() {
		soc.s.Status.StartedAt = metav1.Time{Time: time.Now().UTC()}
		soc.updated = true
	}
	if len(message) > 0 && soc.s.Status.Message != message[0] {
		soc.log.Infof("sensor message %s -> %s", soc.s.Status.Message, message[0])
		soc.s.Status.Message = message[0]
		soc.updated = true
	}

	switch phase {
	case v1alpha1.NodePhaseComplete, v1alpha1.NodePhaseError:
		if markResolved && phaseChange {
			soc.log.Infof("marking sensor complete")
			soc.s.Status.CompletedAt = metav1.Time{Time: time.Now().UTC()}
			if soc.s.ObjectMeta.Labels == nil {
				soc.s.ObjectMeta.Labels = make(map[string]string)
			}
			soc.s.ObjectMeta.Labels[common.LabelKeyComplete] = "true"
			soc.updated = true
		}
	}
}
