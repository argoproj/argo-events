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
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	sensorclientset "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
)

// Factory declares the interface to create real signals from abstract signals
type Factory interface {
	Create(AbstractSignal) (Signal, error)
}

// the registry is a key-value store that holds the various signal factories
type registry struct {
	lock      sync.Mutex
	factories map[string]Factory
}

// ExecutorSession is the session for this sensor job executor
type ExecutorSession struct {
	name            string
	namespace       string
	kubeConfig      *rest.Config
	sensorClientset sensorclientset.Interface
	registry        registry
	log             *zap.Logger
	err             error
}

// New creates a new ExecutorSession
func New(kubeConfig *rest.Config, sensorClientset sensorclientset.Interface, log *zap.Logger) *ExecutorSession {
	return &ExecutorSession{
		kubeConfig:      kubeConfig,
		sensorClientset: sensorClientset,
		registry: registry{
			factories: make(map[string]Factory),
		},
		log: log,
	}
}

// Run the executor
func (es *ExecutorSession) Run(sensor *v1alpha1.Sensor, signalRegisters []func(*ExecutorSession)) error {
	kubeClientset := kubernetes.NewForConfigOrDie(es.kubeConfig)
	es.name = sensor.Name
	es.namespace = sensor.Namespace
	defer es.handleError(kubeClientset, common.CreateJobPrefix(sensor.Name), sensor.Namespace)

	// register the signal factories with this session
	for _, register := range signalRegisters {
		register(es)
	}

	// the channel for all signals to send their events
	events := make(chan Event)
	defer close(events)

	var wg sync.WaitGroup

	// create, initialize, and start the signals
	wg.Add(1)
	go func() {
		for _, rawSignal := range sensor.Spec.Signals {
			var registry Factory
			var ok bool
			if rawSignal.GetType() == v1alpha1.SignalTypeStream {
				registry, ok = es.GetStreamFactory(rawSignal.Stream.Type)
				if !ok {
					es.err = fmt.Errorf("Failed to instantiate %s registry for signal %s", rawSignal.Stream.Type, rawSignal.Name)
					panic(es.err)
				}
			} else {
				registry, ok = es.GetCoreFactory(rawSignal.GetType())
				if !ok {
					es.err = fmt.Errorf("Failed to instantiate %s registry for signal %s", rawSignal.GetType(), rawSignal.Name)
					panic(es.err)
				}
			}

			abstractSignal := AbstractSignal{
				Signal:  rawSignal,
				id:      sensor.NodeID(rawSignal.Name),
				Log:     es.log.With(zap.String("signal", rawSignal.Name)),
				Session: es,
			}
			signal, err := registry.Create(abstractSignal)
			if err != nil {
				es.err = err
				abstractSignal.Log.Panic("failed to create", zap.String("type", string(rawSignal.GetType())), zap.Error(err))
			}

			// start the signals
			err = signal.Start(events)
			if err != nil {
				es.err = err
				abstractSignal.Log.Panic("failed to start", zap.String("type", string(rawSignal.GetType())), zap.Error(err))
			}
		}
		wg.Done()
	}()

	// listen for signal events
	wg.Add(1)
	go func() {
		for event := range events {
			es.log.Info("received event", zap.String("source", event.GetSource()), zap.String("nodeID", event.GetID()))
			stop, resolved := es.syncNode(event)
			if stop {
				err := event.GetSignal().Stop()
				if err != nil {
					//todo: is this worthy of a panic?
					es.log.Warn("failed to stop signal", zap.String("id", event.GetID()))
					continue
				}
				es.log.Info("stopped signal", zap.String("id", event.GetID()))
			}
			if resolved {
				// need to find better way to signal done of events channel
				break
			}
			es.log.Info("node is still running")
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
func (es *ExecutorSession) syncNode(event Event) (bool, bool) {
	ssInterface := es.sensorClientset.ArgoprojV1alpha1().Sensors(es.namespace)
	s, err := ssInterface.Get(es.name, metav1.GetOptions{})
	if err != nil {
		// the sensor was most likely deleted manually - this is a problem, we exit the pod with status 1
		es.log.Error("failed to get sensor on event", zap.Error(err))
	}

	nodeID := event.GetID()
	node, ok := s.Status.Nodes[nodeID]
	if !ok {
		// we should never get here as the job should only be created after the sensor was persisted
		es.log.Warn("node not found or initialized, ignoring event", zap.String("nodeID", event.GetID()))
		return false, s.IsResolved(v1alpha1.NodeTypeSignal)
	}
	// first check that the node is Active - if it's not, we don't want to update the sensor
	if node.Phase != v1alpha1.NodePhaseActive {
		es.log.Warn("node is not active, ignoring event", zap.String("nodeID", event.GetID()))
		return node.IsComplete(), s.IsResolved(v1alpha1.NodeTypeSignal)
	}

	if event.GetError() != nil {
		node.Phase = v1alpha1.NodePhaseError
		node.Message = event.GetError().Error()
	} else {
		node.Phase = v1alpha1.NodePhaseResolved
		node.ResolvedAt = metav1.Time{Time: event.GetTimestamp()}
	}
	s.Status.Nodes[nodeID] = node

	var updated *v1alpha1.Sensor
	for attempt := 0; attempt < 5; attempt++ {
		updated, err = ssInterface.Update(s)
		if err != nil {
			if !errors.IsConflict(err) {
				es.err = err
				es.log.Panic("failed to update sensor's signal node", zap.String("nodeID", event.GetID()), zap.Error(err))
			}
		} else {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return true, updated.IsResolved(v1alpha1.NodeTypeSignal)
}

// AddCoreFactory allows access to add a core signal factory to the session registry
func (es *ExecutorSession) AddCoreFactory(sigType v1alpha1.SignalType, r Factory) {
	es.registry.lock.Lock()
	defer es.registry.lock.Unlock()
	es.registry.factories[string(sigType)] = r
}

// AddStreamFactory allows access to add a stream signal factory to the session registry
func (es *ExecutorSession) AddStreamFactory(streamType string, r Factory) {
	es.registry.lock.Lock()
	defer es.registry.lock.Unlock()
	es.registry.factories[streamType] = r
}

// GetCoreFactory allows access to retrieve a core factory from the session registry
func (es *ExecutorSession) GetCoreFactory(sigType v1alpha1.SignalType) (Factory, bool) {
	es.registry.lock.Lock()
	defer es.registry.lock.Unlock()
	val, ok := es.registry.factories[string(sigType)]
	return val, ok
}

// GetStreamFactory allows access to retrieve a stream factory from the session registry
func (es *ExecutorSession) GetStreamFactory(streamType string) (Factory, bool) {
	es.registry.lock.Lock()
	defer es.registry.lock.Unlock()
	val, ok := es.registry.factories[streamType]
	return val, ok
}

func (es *ExecutorSession) GetKubeConfig() *rest.Config {
	return es.kubeConfig
}
