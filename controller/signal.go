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
	"context"
	"fmt"
	"io"
	"time"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sdk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (soc *sOperationCtx) processSignal(signal v1alpha1.Signal) (*v1alpha1.NodeStatus, error) {
	soc.log.Debugf("evaluating signal '%s'", signal.Name)
	node := soc.getNodeByName(signal.Name)
	if node != nil && node.Phase == v1alpha1.NodePhaseComplete {
		return node, soc.controller.stopSignal(node.ID)
	}

	if node == nil {
		node = soc.initializeNode(signal.Name, v1alpha1.NodeTypeSignal, v1alpha1.NodePhaseNew)
	}

	if node.Phase == v1alpha1.NodePhaseNew {
		if !soc.signalIsPresent(node.ID) {
			// under normal operations, the signal stream is not present when the node is new
			// we now attempt to establish a bi-directional stream with our signal microservice
			// and watch the stream for events.
			err := soc.watchSignal(&signal)
			if err != nil {
				return nil, err
			}
		} else {
			// we get here if the stream already exists even though the node phase is new.
			// this can happen if the sensor was deleted and re-created.
			// let's log a warning but let's keep the stream running to receive events.
			// TODO: add check to see if service is running and we're reading from the stream?
			soc.log.Info("WARNING: signal '%s' is new however signal stream is already present", signal.Name)
		}
	} else {
		if !soc.signalIsPresent(node.ID) {
			// we get here if the signal is in active or error phase and the signal stream is missing.
			// this means that we had down-time for this signal - we could have missed events.
			// this can happen if the controller or a signal pod serving the stream goes down.
			// let's log a warning and attempt to re-establish a stream and watch for events.
			soc.log.Infof("WARNING: event stream for signal '%s' is missing - could have missed events! reconnecting stream...", signal.Name)
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

	return soc.markNodePhase(signal.Name, v1alpha1.NodePhaseActive, "stream established"), nil
}

// checks to see if the signal is present
// TODO: include a check on the stream interface to check if stream is still open?
func (soc *sOperationCtx) signalIsPresent(nodeID string) bool {
	soc.controller.signalMu.Lock()
	_, ok := soc.controller.signalStreams[nodeID]
	soc.controller.signalMu.Unlock()
	return ok
}

// resolveClient is a helper method to find the correct SignalClient for this signal
func (soc *sOperationCtx) resolveClient(signal *v1alpha1.Signal) (sdk.SignalClient, error) {
	var typ string
	switch signal.GetType() {
	case v1alpha1.SignalTypeStream:
		typ = signal.Stream.Type
	default:
		typ = string(signal.GetType())
	}
	client, err := soc.controller.signalMgr.Dispense(typ)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// adds a new signal to the controller's signals
func (soc *sOperationCtx) watchSignal(signal *v1alpha1.Signal) error {
	client, err := soc.resolveClient(signal)
	if err != nil {
		return err
	}

	// create the context for this stream
	ctx := context.Background()
	var cancel context.CancelFunc
	if signal.Deadline > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(signal.Deadline))
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}

	stream, err := client.Listen(ctx, signal)
	if err != nil {
		cancel()
		return err
	}
	nodeID := soc.s.NodeID(signal.Name)

	streamCtx := streamCtx{
		ctx:    ctx,
		cancel: cancel,
		sensor: soc.s.Name,
		nodeID: nodeID,
		signal: signal,
		stream: stream,
	}

	soc.controller.signalMu.Lock()
	soc.controller.signalStreams[nodeID] = stream
	soc.controller.signalMu.Unlock()

	go soc.controller.listenOnStream(&streamCtx)
	return nil
}

// stop the signal by:
// 1. deleting the stream from the controller's signalStreams map
// 2. sending the terminate signal on the stream and close it
// NOTE: this is a method on the controller
func (c *SensorController) stopSignal(nodeID string) error {
	c.signalMu.Lock()
	stream, ok := c.signalStreams[nodeID]
	c.signalStreams[nodeID] = nil
	delete(c.signalStreams, nodeID)
	c.signalMu.Unlock()
	if ok && stream != nil {
		if err := stream.Send(sdk.Terminate); err != nil {
			return err
		}
		if err := stream.Close(); err != nil {
			return err
		}
	}
	return nil
}

// streamCtx contains the relevant context data for processing the signal event stream
type streamCtx struct {
	ctx    context.Context
	cancel context.CancelFunc
	sensor string
	nodeID string
	signal *v1alpha1.Signal
	stream sdk.SignalService_ListenService
}

// listens for events on the event stream. meant to be run as a separate goroutine
// this will terminate once the stream receives an EOF indicating it has completed or it encounters a stream error.
// NOTE: this is a method on the controller
func (c *SensorController) listenOnStream(streamCtx *streamCtx) {
	// TODO: possible context leak if we don't utilize cancel
	//defer cancel()
	sensors := c.sensorClientset.ArgoprojV1alpha1().Sensors(c.Config.Namespace)
	for {
		in, streamErr := streamCtx.stream.Recv()
		if streamErr == io.EOF {
			return
		}

		// retrieve the sensor & node as this will be needed to update accordingly
		s, err := sensors.Get(streamCtx.sensor, metav1.GetOptions{})
		if err != nil {
			// we can get here if the sensor was deleted and the stream was not closed
			// we should attempt to log a warning and stop & close the signal stream
			// TODO: why call stopSignal() when we have access to the stream within this func?
			c.log.Warnf("Event Stream (%s/%s) Error: %s. Terminating event stream...", streamCtx.sensor, streamCtx.signal.Name, err)
			err := c.stopSignal(streamCtx.nodeID)
			if err != nil {
				c.log.Panicf("failed to stop signal stream '%s': %s", streamCtx.nodeID, err)
			}
		}
		phase := s.Status.Phase
		msg := s.Status.Message
		node, ok := s.Status.Nodes[streamCtx.nodeID]
		if !ok {
			// TODO: should we re-initialize this node?
			c.log.Panicf("Event Stream (%s/%s) Fatal: '%s' node is missing from sensor's nodes", streamCtx.sensor, streamCtx.signal.Name, streamCtx.nodeID)
		}

		if streamErr != nil {
			c.log.Infof("Event Stream (%s/%s) Error: removing & terminating stream due to: %s", streamCtx.sensor, streamCtx.signal.Name, streamErr)
			// error received from the stream
			// remove the stream from the signalStreams map
			c.signalMu.Lock()
			c.signalStreams[streamCtx.nodeID] = nil
			delete(c.signalStreams, streamCtx.nodeID)
			c.signalMu.Unlock()

			// mark the sensor & node as error phase
			phase = v1alpha1.NodePhaseError
			msg = fmt.Sprintf("Event Stream encountered err: %s", streamErr)
			node.Phase = v1alpha1.NodePhaseError
			node.Message = streamErr.Error()
		} else {
			ok, err := filterEvent(streamCtx.signal.Filters, in.Event)
			if err != nil {
				c.log.Infof("Event Stream (%s/%s) Msg: (Action:IGNORED) - Failed to filter event: %s", streamCtx.sensor, streamCtx.signal.Name, err)
				continue
			}
			if ok {
				c.log.Debugf("Event Stream (%s/%s) Msg: (Action:ACCEPTED) - Context: %s", streamCtx.sensor, streamCtx.signal.Name, in.Event.Context)
				node.LatestEvent = &v1alpha1.EventWrapper{Event: *in.Event}
			} else {
				c.log.Debugf("Event Stream (%s/%s) Msg: (Action:FILTERED) - Context: %s", streamCtx.sensor, streamCtx.signal.Name, in.Event.Context)
				continue
			}
		}

		// TODO: perform a Patch here instead as the sensor could become stale
		// and this exponential backoff is slow
		err = wait.ExponentialBackoff(common.DefaultRetry, func() (bool, error) {
			s, err := sensors.Get(streamCtx.sensor, metav1.GetOptions{})
			if err != nil {
				return false, err
			}
			s.Status.Nodes[streamCtx.nodeID] = node
			s.Status.Phase = phase
			s.Status.Message = msg
			_, err = sensors.Update(s)
			if err != nil {
				if !common.IsRetryableKubeAPIError(err) {
					return false, err
				}
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			c.log.Panicf("Event Stream (%s/%s) Update Resource Failed: %s", streamCtx.sensor, streamCtx.signal.Name, err)
		}

		// finally check if there was a streamErr, we must return
		if streamErr != nil {
			return
		}
	}
}
