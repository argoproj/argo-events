package controller

import (
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// signalCtx is the context for handling signal
type signalCtx struct {
	nodename string
	obj      *v1alpha1.Signal
	plugin   shared.Signaler
}

func (soc *sOperationCtx) processSignal(signal v1alpha1.Signal) (*v1alpha1.NodeStatus, error) {
	soc.log.Debugf("evaluating signal '%s'", signal.Name)
	node := soc.getNodeByName(signal.Name)
	if node != nil && node.IsComplete() {
		soc.stopSignal(&signal)
		return node, nil
	}

	if node == nil {
		node = soc.initializeNode(signal.Name, v1alpha1.NodeTypeSignal, v1alpha1.NodePhaseNew)
	}

	// if signal is new (phase=new) == signal not present (signaler=nil)
	if node.Phase == v1alpha1.NodePhaseNew {
		// can the node phase and the status of the signal watch ever introduce discrepency?
		if !soc.signalIsPresent(signal.Name) {
			err := soc.watchSignal(&signal)
			if err != nil {
				return nil, err
			}
		} else {
			// we get here if the phase == New and the signaler already exists -- should never happen...
			// in any case, let's log a warning as this is unintended but let's keep the signaler running to receive events
			// TODO: add check to see if plugin is running?
			soc.log.Warn("signal is new however signal plugin is already present")
		}
	} else {
		if !soc.signalIsPresent(signal.Name) {
			// signal is active but signaler plugin is missing... meaning we could have been missing events for this signal
			// we could get here if the controller crashes...
			// plugin may have crashed?
			soc.log.Warn("signal is active however signal plugin is missing, potentially could have missed events! attempting to restart plugin.")
			err := soc.watchSignal(&signal)
			if err != nil {
				return nil, err
			}
		}
	}

	// let's check the latest event to see if node has completed?
	if node.LatestEvent != nil {
		soc.events[node.ID] = node.LatestEvent.Event
		if !node.LatestEvent.Seen {
			soc.s.Status.Nodes[node.ID].LatestEvent.Seen = true
			soc.updated = true
		}
		return soc.markNodePhase(signal.Name, v1alpha1.NodePhaseComplete), nil
	}

	return soc.markNodePhase(signal.Name, v1alpha1.NodePhaseActive), nil
}

// checks to see if the signal is present
// TODO: include a check on the signaler interface to check if signal is active?
func (soc *sOperationCtx) signalIsPresent(name string) bool {
	nodeID := soc.s.NodeID(name)
	soc.controller.signalMu.Lock()
	_, ok := soc.controller.signals[nodeID]
	soc.controller.signalMu.Unlock()
	return ok
}

// adds a new signal to the controller's signals
func (soc *sOperationCtx) watchSignal(signal *v1alpha1.Signal) error {
	plugin, err := soc.controller.signalProto.Dispense(signal.GetType())
	if err != nil {
		return err
	}
	signaler := plugin.(shared.Signaler)

	events, err := signaler.Start(signal)
	if err != nil {
		return err
	}
	nodeID := soc.s.NodeID(signal.Name)
	soc.controller.signalMu.Lock()
	soc.controller.signals[nodeID] = signaler
	soc.controller.signalMu.Unlock()

	go soc.controller.listenForEvents(soc.s.Name, signal.Name, nodeID, events)
	return nil
}

func (soc *sOperationCtx) stopSignal(signal *v1alpha1.Signal) {
	nodeID := soc.s.NodeID(signal.Name)
	soc.controller.signalMu.Lock()
	signaler, ok := soc.controller.signals[nodeID]
	soc.controller.signals[nodeID] = nil
	soc.controller.signalMu.Unlock()
	if ok && signaler != nil {
		if err := signaler.Stop(); err != nil {
			soc.log.Warnf("failed to stop the signaler '%s'. cause: %s", nodeID, err)
		}
	}
}

// listens for events on the events channel
// meant to be run as a separate goroutine
// this will terminate once the events chan is closed
// the events chan should be closed by the future operations on this sensor
func (c *SensorController) listenForEvents(sensor, signal, nodeID string, events <-chan *v1alpha1.Event) {
	sensors := c.sensorClientset.ArgoprojV1alpha1().Sensors(c.Config.Namespace)
	for event := range events {
		c.log.Infof("sensor '%s' received event for signal '%s'", sensor, signal)
		// todo: store the event in an external store

		s, err := sensors.Get(sensor, metav1.GetOptions{})
		if err != nil {
			// we can get here if the sensor was removed after completion or deleted and the signaler was not stopped
			// we should attempt to log a warning and stop the signaler
			c.log.Warnf("sensor '%s' cannot be found due to: %s. Stopping signaler...", sensor, err)
			c.signalMu.Lock()
			defer c.signalMu.Unlock()
			signaler, ok := c.signals[nodeID]
			if !ok {
				c.log.Panicf("signaler '%s' not found, unable to stop listenForEvents() gracefully", nodeID)
			}
			err := signaler.Stop()
			if err != nil {
				c.log.Panicf("failed to stop signaler '%s'", nodeID)
			}
		}
		// requeue this sensor with the event context
		/*
			key, err := cache.MetaNamespaceKeyFunc(s)
			if err == nil {
				c.queue.Add(key)
			}
		*/
		node, ok := s.Status.Nodes[nodeID]
		if !ok {
			c.log.Panicf("'%s' node is missing from sensor's nodes", nodeID)
		}
		node.LatestEvent = &v1alpha1.EventWrapper{Event: *event}
		s.Status.Nodes[nodeID] = node
		_, err = sensors.Update(s)
		if err != nil {
			c.log.Panicf("failed to update sensor with event")
		}
	}
}
