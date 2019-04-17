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

package gateways

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/gateway"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	corev1 "k8s.io/api/core/v1"
)

// createInternalEventSources creates an internal representation of event source declared in the gateway configmap.
// returned event sources are map of hash of event source and event source itself.
// Creating a hash of event source makes it easy to check equality of two event sources.
func (gc *GatewayConfig) createInternalEventSources(cm *corev1.ConfigMap) (map[string]*EventSourceContext, error) {
	configs := make(map[string]*EventSourceContext)
	for configKey, configValue := range cm.Data {
		hashKey := common.Hasher(configKey + configValue)
		gc.Log.WithFields(
			map[string]interface{}{
				"config-key":   configKey,
				"config-value": configValue,
				"hash":         string(hashKey),
			},
		).Info("event source")

		// create a connection to gateway server
		ctx, cancel := context.WithCancel(context.Background())
		conn, err := grpc.Dial(
			fmt.Sprintf("localhost:%s", gc.serverPort),
			grpc.WithBlock(),
			grpc.WithInsecure(),
			grpc.WithTimeout(common.ServerConnTimeout*time.Second))
		if err != nil {
			gc.Log.WithError(err).
				WithField("conn-state", conn.GetState().String()).Panic("failed to connect to gateway server")
			cancel()
			return nil, err
		}

		gc.Log.WithField("state", conn.GetState().String()).Info("state of the connection")

		configs[hashKey] = &EventSourceContext{
			Data: &EventSourceData{
				ID:     hashKey,
				Src:    configKey,
				Config: configValue,
			},
			Cancel: cancel,
			Ctx:    ctx,
			Client: NewEventingClient(conn),
			Conn:   conn,
		}
	}
	return configs, nil
}

// diffConfig diffs currently registered event sources and the event sources in the gateway configmap
// It simply matches the event source strings. So, if event source string differs through some sequence of definition
// and although the event sources are actually same, this method will treat them as different event sources.
// retunrs staleConfig - event sources to be removed from gateway
// newConfig - new event sources to run
func (gc *GatewayConfig) diffEventSources(newConfigs map[string]*EventSourceContext) (staleConfigKeys []string, newConfigKeys []string) {
	var currentConfigKeys []string
	var updatedConfigKeys []string

	for currentConfigKey := range gc.registeredConfigs {
		currentConfigKeys = append(currentConfigKeys, currentConfigKey)
	}
	for updatedConfigKey := range newConfigs {
		updatedConfigKeys = append(updatedConfigKeys, updatedConfigKey)
	}

	gc.Log.WithField("current-event-sources-keys", currentConfigKeys).Debug("event sources hashes")
	gc.Log.WithField("updated-event-sources-keys", updatedConfigKeys).Debug("event sources hashes")

	swapped := false
	// iterates over current event sources and updated event sources
	// and creates two arrays, first one containing event sources that need to removed
	// and second containing new event sources that need to be added and run.
	for i := 0; i < 2; i++ {
		for _, cc := range currentConfigKeys {
			found := false
			for _, uc := range updatedConfigKeys {
				if cc == uc {
					found = true
					break
				}
			}
			if !found {
				if swapped {
					newConfigKeys = append(newConfigKeys, cc)
				} else {
					staleConfigKeys = append(staleConfigKeys, cc)
				}
			}
		}
		if i == 0 {
			currentConfigKeys, updatedConfigKeys = updatedConfigKeys, currentConfigKeys
			swapped = true
		}
	}
	return
}

// startEventSources starts new event sources added to gateway
func (gc *GatewayConfig) startEventSources(eventSources map[string]*EventSourceContext, keys []string) {
	for _, key := range keys {
		eventSource := eventSources[key]
		// register the event source
		gc.registeredConfigs[key] = eventSource

		log := gc.Log.WithField(common.LabelEventSource, eventSource.Data.Src)

		log.Info("activating new event source")

		go func() {
			// conn should be in READY state
			if eventSource.Conn.GetState() != connectivity.Ready {
				gc.Log.Error("connection is not in ready state.")
				gc.StatusCh <- EventSourceStatus{
					Phase:   v1alpha1.NodePhaseError,
					Id:      eventSource.Data.ID,
					Message: "connection_is_not_in_ready_state",
					Name:    eventSource.Data.Src,
				}
				return
			}

			// validate event source
			if valid, _ := eventSource.Client.ValidateEventSource(eventSource.Ctx, &EventSource{
				Data: eventSource.Data.Config,
				Name: eventSource.Data.Src,
				Id:   eventSource.Data.ID,
			}); !valid.IsValid {
				gc.Log.WithFields(
					map[string]interface{}{
						"validation-failure": valid.Reason,
					},
				).Error("event source is not valid")
				if err := eventSource.Conn.Close(); err != nil {
					gc.Log.WithError(err).Error("failed to close client connection")
				}
				gc.StatusCh <- EventSourceStatus{
					Phase:   v1alpha1.NodePhaseError,
					Id:      eventSource.Data.ID,
					Message: "event_source_is_not_valid",
					Name:    eventSource.Data.Src,
				}
				return
			}

			gc.Log.Info("event source is valid")

			// mark event source as running
			gc.StatusCh <- EventSourceStatus{
				Phase:   v1alpha1.NodePhaseRunning,
				Message: "event_source_is_running",
				Id:      eventSource.Data.ID,
				Name:    eventSource.Data.Src,
			}

			// listen to events from gateway server
			eventStream, err := eventSource.Client.StartEventSource(eventSource.Ctx, &EventSource{
				Name: eventSource.Data.Src,
				Data: eventSource.Data.Config,
				Id:   eventSource.Data.ID,
			})
			if err != nil {
				gc.Log.WithError(err).Error("error occurred while starting event source")
				gc.StatusCh <- EventSourceStatus{
					Phase:   v1alpha1.NodePhaseError,
					Message: "failed_to_receive_event_stream",
					Name:    eventSource.Data.Src,
					Id:      eventSource.Data.ID,
				}
				return
			}

			gc.Log.Info("started listening to events from gateway server")
			for {
				event, err := eventStream.Recv()
				if err != nil {
					if err == io.EOF {
						gc.Log.Info("event source has stopped")
						gc.StatusCh <- EventSourceStatus{
							Phase:   v1alpha1.NodePhaseCompleted,
							Message: "event_source_has_been_stopped",
							Name:    eventSource.Data.Src,
							Id:      eventSource.Data.ID,
						}
						return
					}

					gc.Log.WithError(err).Error("failed to receive event from stream")
					gc.StatusCh <- EventSourceStatus{
						Phase:   v1alpha1.NodePhaseError,
						Message: "failed_to_receive_event_from_event_source_stream",
						Name:    eventSource.Data.Src,
						Id:      eventSource.Data.ID,
					}
					return
				}
				err = gc.DispatchEvent(event)
				if err != nil {
					// escalate error through a K8s event
					labels := map[string]string{
						common.LabelEventType:              string(common.EscalationEventType),
						common.LabelGatewayEventSourceName: eventSource.Data.Src,
						common.LabelGatewayName:            gc.Name,
						common.LabelGatewayEventSourceID:   eventSource.Data.ID,
						common.LabelOperation:              "dispatch_event_to_watchers",
					}
					if err := common.GenerateK8sEvent(gc.Clientset, fmt.Sprintf("failed to dispatch event to watchers"), common.EscalationEventType, "event dispatch failed", gc.Name, gc.Namespace, gc.controllerInstanceID, gateway.Kind, labels); err != nil {
						gc.Log.WithError(err).Error("failed to create K8s event to escalate event dispatch failure")
					}
					gc.Log.WithError(err).Error("failed to dispatch event to watchers")
				}
			}
		}()
	}
}

// stopEventSources stops an existing event sources
func (gc *GatewayConfig) stopEventSources(configs []string) {
	for _, configKey := range configs {
		eventSource := gc.registeredConfigs[configKey]
		delete(gc.registeredConfigs, configKey)
		gc.Log.WithEventSource(eventSource.Data.Src).Info("removing the event source")
		gc.StatusCh <- EventSourceStatus{
			Phase:   v1alpha1.NodePhaseRemove,
			Id:      eventSource.Data.ID,
			Message: "event_source_is_removed",
			Name:    eventSource.Data.Src,
		}
		eventSource.Cancel()
		if err := eventSource.Conn.Close(); err != nil {
			gc.Log.WithEventSource(eventSource.Data.Src).WithError(err).Error("failed to close client connection")
		}
	}
}

// manageEventSources syncs registered event sources and updated gateway configmap
func (gc *GatewayConfig) manageEventSources(cm *corev1.ConfigMap) error {
	eventSources, err := gc.createInternalEventSources(cm)
	if err != nil {
		return err
	}

	staleEventSources, newEventSources := gc.diffEventSources(eventSources)
	gc.Log.WithField(common.LabelEventSource, staleEventSources).Info("stale event sources")
	gc.Log.WithField(common.LabelEventSource, newEventSources).Info("new event sources")

	// stop existing event sources
	gc.stopEventSources(staleEventSources)

	// start new event sources
	gc.startEventSources(eventSources, newEventSources)

	return nil
}
