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

package job

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/job/shared"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensorclientset "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	"github.com/golang/protobuf/ptypes"
	plugin "github.com/hashicorp/go-plugin"
)

// ExecutorSession is the session for this sensor job executor
type ExecutorSession struct {
	name            string
	namespace       string
	kubeConfig      *rest.Config
	sensorClientset sensorclientset.Interface
	log             *zap.Logger
	err             error
}

// New creates a new ExecutorSession
func New(kubeConfig *rest.Config, sensorClientset sensorclientset.Interface, log *zap.Logger) *ExecutorSession {
	return &ExecutorSession{
		kubeConfig:      kubeConfig,
		sensorClientset: sensorClientset,
		log:             log,
	}
}

// Run the executor
func (es *ExecutorSession) Run(sensor *v1alpha1.Sensor, plugins plugin.ClientProtocol) error {
	kubeClientset := kubernetes.NewForConfigOrDie(es.kubeConfig)
	es.name = sensor.Name
	es.namespace = sensor.Namespace
	defer es.handleError(kubeClientset, common.CreateJobPrefix(sensor.Name), sensor.Namespace)

	// the channel for all signals to send their events
	events := make(chan shared.Event)
	defer close(events)

	var wg sync.WaitGroup

	// start the signals
	wg.Add(1)
	go func() {
		for _, rawSignal := range sensor.Spec.Signals {
			var name string
			if rawSignal.GetType() == v1alpha1.SignalTypeStream {
				name = rawSignal.Stream.Type
			} else {
				name = string(rawSignal.GetType())
			}
			raw, err := plugins.Dispense(name)
			if err != nil {
				es.err = err
				es.log.Panic("failed to dispense signal plugin", zap.String("name", name), zap.Error(err))
			}
			signaler := raw.(shared.Signaler)

			sigEvents, err := signaler.Start(&rawSignal)
			if err != nil {
				es.err = err
				es.log.Panic("failed to start signal", zap.Error(err))
			}

			// multiplex the sigEvents into events? or just range over sigEvents?
			wg.Add(1)
			go func() {
				for event := range sigEvents {
					es.log.Info("received event")
					// todo: add signal processing and sync node
					es.syncNode(sensor.NodeID(rawSignal.Name), event)

					err := signaler.Stop()
					if err != nil {
						es.log.Warn("failed to stop signal", zap.Error(err))
					}
				}
				wg.Done()
			}()
		}
		wg.Done()
	}()

	wg.Wait()
	err := common.AddJobLabel(kubeClientset, common.CreateJobPrefix(sensor.Name), es.namespace, common.LabelKeyResolved, "true")
	if err != nil {
		es.log.Warn("failed to mark job as resolved")
	}
	es.log.Info("successfully resolved all signals; executor terminating")
	return nil
}

// handleError annotates the job with an error message in the job's annotations
func (es *ExecutorSession) handleError(kubeClient kubernetes.Interface, name, namespace string) {
	if r := recover(); r != nil {
		_ = common.AddJobAnnotation(kubeClient, name, namespace, common.AnnotationKeyNodeMessage, fmt.Sprintf("%v", r))
		es.log.Fatal("executor panic", zap.Stack("stackstrace"))
	} else {
		if es.err != nil {
			_ = common.AddJobAnnotation(kubeClient, name, namespace, common.AnnotationKeyNodeMessage, es.err.Error())
		}
	}
}

// syncNode synchronizes the sensor resource with an event update to the signal node
// returns a tuple of booleans
// first value is if we should stop the signal
// second value is if the sensor is completely resolved
func (es *ExecutorSession) syncNode(nodeID string, event shared.Event) (bool, bool) {
	log := es.log.With(zap.String("nodeID", nodeID))
	ssInterface := es.sensorClientset.ArgoprojV1alpha1().Sensors(es.namespace)
	s, err := ssInterface.Get(es.name, metav1.GetOptions{})
	if err != nil {
		// the sensor was most likely deleted manually - this is a problem, we exit the pod with status 1
		log.Error("failed to get sensor on event", zap.Error(err))
	}

	node, ok := s.Status.Nodes[nodeID]
	if !ok {
		// we should never get here as the job should only be created after the sensor was persisted
		log.Warn("node not found or initialized, ignoring event")
		return false, s.IsResolved(v1alpha1.NodeTypeSignal)
	}
	// first check that the node is Active - if it's not, we don't want to update the sensor
	if node.Phase != v1alpha1.NodePhaseActive {
		log.Warn("node is not active, ignoring event")
		return node.IsComplete(), s.IsResolved(v1alpha1.NodeTypeSignal)
	}

	t, err := ptypes.Timestamp(event.Context.EventTime)
	if err != nil {
		log.Warn("event context is missing EventTime")
	}

	node.Phase = v1alpha1.NodePhaseResolved
	node.ResolvedAt = metav1.Time{Time: t}
	s.Status.Nodes[nodeID] = node

	var updated *v1alpha1.Sensor
	for attempt := 0; attempt < 5; attempt++ {
		updated, err = ssInterface.Update(s)
		if err != nil {
			if !errors.IsConflict(err) {
				es.err = err
				log.Panic("failed to update sensor's signal node", zap.Error(err))
			}
		} else {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return true, updated.IsResolved(v1alpha1.NodeTypeSignal)
}

func (es *ExecutorSession) GetKubeConfig() *rest.Config {
	return es.kubeConfig
}
