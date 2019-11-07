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

package main

import (
	"context"
	"fmt"
	"github.com/argoproj/argo-events/gateways"
	"github.com/pkg/errors"
	"io"
	"time"

	"github.com/argoproj/argo-events/common"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	eventSourceV1Alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/gateway"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// setupInternalEventSources sets up an internal representation of event sources within given EventSource resource
func (gatewayCfg *GatewayConfig) setupInternalEventSources(eventSourceName string, eventSourceValue interface{}, internalEventSources map[string]*EventSourceContext) {
	value, err := yaml.Marshal(eventSourceValue)
	if err != nil {
		gatewayCfg.logger.WithField("event-source-name", eventSourceName).Errorln("failed to marshal the event source value, won't process it")
		return
	}

	hashKey := common.Hasher(eventSourceName + string(value))

	logger := gatewayCfg.logger.WithFields(
		map[string]interface{}{
			"event-source-name":  eventSourceName,
			"event-source-value": value,
		},
	)

	logger.WithField("hash", string(hashKey)).Debugln("hash of the event source")

	// create a connection to gateway server
	ctx, cancel := context.WithCancel(context.Background())
	conn, err := grpc.Dial(
		fmt.Sprintf("localhost:%s", gatewayCfg.serverPort),
		grpc.WithBlock(),
		grpc.WithInsecure(),
		grpc.WithTimeout(common.ServerConnTimeout*time.Second))
	if err != nil {
		logger.WithError(err).Errorln("failed to connect to gateway server")
		cancel()
	}

	logger.WithField("state", conn.GetState().String()).Info("state of the connection to gateway server")

	internalEventSources[hashKey] = &EventSourceContext{
		source: &gateways.EventSource{
			Id:      hashKey,
			Name:    eventSourceName,
			Value:   value,
			Version: gatewayCfg.gateway.Spec.Version,
			Type:    string(gatewayCfg.gateway.Spec.Type),
		},
		cancel: cancel,
		ctx:    ctx,
		client: gateways.NewEventingClient(conn),
		conn:   conn,
	}
}

// createInternalEventSources creates an internal representation of a event-source.
// returned event sources are map of hash of event source and event source itself.
// Creating a hash of event source makes it easy to check equality of two event sources.
func (gatewayCfg *GatewayConfig) createInternalEventSources(eventSource *eventSourceV1Alpha1.EventSource) (map[string]*EventSourceContext, error) {
	internalEventSources := make(map[string]*EventSourceContext)

	switch gatewayCfg.gateway.Spec.Type {
	case apicommon.SNSEvent:
		for key, value := range eventSource.Spec.SNS {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.SQSEvent:
		for key, value := range eventSource.Spec.SQS {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.PubSubEvent:
		for key, value := range eventSource.Spec.PubSub {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.NATSEvent:
		for key, value := range eventSource.Spec.NATS {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.FileEvent:
		for key, value := range eventSource.Spec.File {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.CalendarEvent:
		for key, value := range eventSource.Spec.Calendar {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.AMQPEvent:
		for key, value := range eventSource.Spec.AMQP {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.GitHubEvent:
		for key, value := range eventSource.Spec.Github {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.GitLabEvent:
		for key, value := range eventSource.Spec.Gitlab {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.HDFSEvent:
		for key, value := range eventSource.Spec.HDFS {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.KafkaEvent:
		for key, value := range eventSource.Spec.Kafka {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.MinioEvent:
		for key, value := range eventSource.Spec.Minio {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.MQTTEvent:
		for key, value := range eventSource.Spec.MQTT {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.ResourceEvent:
		for key, value := range eventSource.Spec.Resource {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.SlackEvent:
		for key, value := range eventSource.Spec.Slack {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.StorageGridEvent:
		for key, value := range eventSource.Spec.StorageGrid {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	case apicommon.WebhookEvent:
		for key, value := range eventSource.Spec.Webhook {
			gatewayCfg.setupInternalEventSources(key, value, internalEventSources)
		}
	}

	return internalEventSources, nil
}

// diffConfig diffs currently active event sources and updated event sources.
// It simply matches the event source strings. So, if event source string differs through some sequence of definition
// and although the event sources are actually same, this method will treat them as different event sources.
// retunrs staleConfig - event sources to be removed from gateway
// newConfig - new event sources to run
func (gatewayCfg *GatewayConfig) diffEventSources(newConfigs map[string]*EventSourceContext) (staleConfigKeys []string, newConfigKeys []string) {
	var currentConfigKeys []string
	var updatedConfigKeys []string

	for currentConfigKey := range gatewayCfg.registeredConfigs {
		currentConfigKeys = append(currentConfigKeys, currentConfigKey)
	}
	for updatedConfigKey := range newConfigs {
		updatedConfigKeys = append(updatedConfigKeys, updatedConfigKey)
	}

	gatewayCfg.logger.WithField("current-event-sources-keys", currentConfigKeys).Debugln("event sources hashes")
	gatewayCfg.logger.WithField("updated-event-sources-keys", updatedConfigKeys).Debugln("event sources hashes")

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
func (gatewayCfg *GatewayConfig) startEventSources(eventSources map[string]*EventSourceContext, keys []string) {
	for _, key := range keys {
		eventSource := eventSources[key]
		// register the event source
		gatewayCfg.registeredConfigs[key] = eventSource

		logger := gatewayCfg.logger.WithField(common.LabelEventSource, eventSource.source.Name)

		logger.Infoln("activating new event source...")

		go func() {
			// conn should be in READY state
			if eventSource.conn.GetState() != connectivity.Ready {
				logger.Errorln("connection is not in ready state.")
				gatewayCfg.statusCh <- EventSourceStatus{
					Phase:   v1alpha1.NodePhaseError,
					Id:      eventSource.source.Id,
					Message: "connection_is_not_in_ready_state",
					Name:    eventSource.source.Name,
				}
				return
			}

			// validate event source
			if valid, _ := eventSource.client.ValidateEventSource(eventSource.ctx, eventSource.source); !valid.IsValid {
				logger.WithFields(
					map[string]interface{}{
						"validation-failure": valid.Reason,
					},
				).Errorln("event source is not valid")
				if err := eventSource.conn.Close(); err != nil {
					logger.WithError(err).Errorln("failed to close client connection")
				}
				gatewayCfg.statusCh <- EventSourceStatus{
					Phase:   v1alpha1.NodePhaseError,
					Id:      eventSource.source.Id,
					Message: "event_source_is_not_valid",
					Name:    eventSource.source.Name,
				}
				return
			}

			logger.Infoln("event source is valid")

			// mark event source as running
			gatewayCfg.statusCh <- EventSourceStatus{
				Phase:   v1alpha1.NodePhaseRunning,
				Message: "event_source_is_running",
				Id:      eventSource.source.Id,
				Name:    eventSource.source.Name,
			}

			// listen to events from gateway server
			eventStream, err := eventSource.client.StartEventSource(eventSource.ctx, eventSource.source)
			if err != nil {
				logger.WithError(err).Errorln("error occurred while starting event source")
				gatewayCfg.statusCh <- EventSourceStatus{
					Phase:   v1alpha1.NodePhaseError,
					Message: "failed_to_receive_event_stream",
					Name:    eventSource.source.Name,
					Id:      eventSource.source.Id,
				}
				return
			}

			logger.Infoln("listening to events from gateway server...")
			for {
				event, err := eventStream.Recv()
				if err != nil {
					if err == io.EOF {
						logger.Infoln("event source has stopped")
						gatewayCfg.statusCh <- EventSourceStatus{
							Phase:   v1alpha1.NodePhaseCompleted,
							Message: "event_source_has_been_stopped",
							Name:    eventSource.source.Name,
							Id:      eventSource.source.Id,
						}
						return
					}

					logger.WithError(err).Errorln("failed to receive event from stream")
					gatewayCfg.statusCh <- EventSourceStatus{
						Phase:   v1alpha1.NodePhaseError,
						Message: "failed_to_receive_event_from_event_source_stream",
						Name:    eventSource.source.Name,
						Id:      eventSource.source.Id,
					}
					return
				}
				err = gatewayCfg.DispatchEvent(event)
				if err != nil {
					// escalate error through a K8s event
					labels := map[string]string{
						common.LabelEventType:              string(common.EscalationEventType),
						common.LabelGatewayEventSourceName: eventSource.source.Name,
						common.LabelGatewayName:            gatewayCfg.name,
						common.LabelGatewayEventSourceID:   eventSource.source.Id,
						common.LabelOperation:              "dispatch_event_to_watchers",
					}
					if err := common.GenerateK8sEvent(gatewayCfg.k8sClient, fmt.Sprintf("failed to dispatch event to watchers"), common.EscalationEventType, "event dispatch failed", gatewayCfg.name, gatewayCfg.namespace, gatewayCfg.controllerInstanceID, gateway.Kind, labels); err != nil {
						logger.WithError(err).Errorln("failed to create K8s event to escalate event dispatch failure")
					}
					logger.WithError(err).Errorln("failed to dispatch event to watchers")
				}
			}
		}()
	}
}

// stopEventSources stops an existing event sources
func (gatewayCfg *GatewayConfig) stopEventSources(eventSourceNames []string) {
	for _, eventSourceName := range eventSourceNames {
		eventSource := gatewayCfg.registeredConfigs[eventSourceName]

		logger := gatewayCfg.logger.WithField(common.LabelEventSource, eventSourceName)

		logger.Infoln("deleting event source from internal store...")
		delete(gatewayCfg.registeredConfigs, eventSourceName)

		logger.WithField(common.LabelEventSource, eventSource.source.Name).Infoln("stopping the event source")
		gatewayCfg.statusCh <- EventSourceStatus{
			Phase:   v1alpha1.NodePhaseRemove,
			Id:      eventSource.source.Id,
			Message: "event_source_is_removed",
			Name:    eventSource.source.Name,
		}
		eventSource.cancel()
		if err := eventSource.conn.Close(); err != nil {
			logger.WithField(common.LabelEventSource, eventSource.source.Name).WithError(err).Errorln("failed to close client connection")
		}
	}
}

// manageEventSources syncs active event-sources and the updated ones
func (gatewayCfg *GatewayConfig) manageEventSources(eventSource *eventSourceV1Alpha1.EventSource) error {
	if gatewayCfg.gateway.Spec.Version != eventSource.Spec.Version {
		return errors.Errorf("gateway and event-source version mismatch. gateway-version: %s, event-source-version: %s", gatewayCfg.gateway.Spec.Version, eventSource.Spec.Version)
	}

	internalEventSources, err := gatewayCfg.createInternalEventSources(eventSource)
	if err != nil {
		return err
	}

	staleEventSources, newEventSources := gatewayCfg.diffEventSources(internalEventSources)
	gatewayCfg.logger.WithField(common.LabelEventSource, staleEventSources).Infoln("stale event sources")
	gatewayCfg.logger.WithField(common.LabelEventSource, newEventSources).Infoln("new event sources")

	// stop existing event sources
	gatewayCfg.stopEventSources(staleEventSources)

	// start new event sources
	gatewayCfg.startEventSources(internalEventSources, newEventSources)

	return nil
}
