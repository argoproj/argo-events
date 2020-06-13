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
	"io"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	eventSourceV1Alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// populateEventSourceContexts sets up the contexts for event sources
func (gatewayContext *GatewayContext) populateEventSourceContexts(name string, value interface{}, eventSourceContexts map[string]*EventSourceContext) {
	body, err := yaml.Marshal(value)
	if err != nil {
		gatewayContext.logger.WithField("event-source-name", name).Errorln("failed to marshal the event source value, won't process it")
		return
	}

	hashKey := common.Hasher(name + string(body))

	logger := gatewayContext.logger.WithFields(logrus.Fields{
		"name":  name,
		"value": value,
	})

	logger.WithField("hash", hashKey).Debugln("hash of the event source")

	// create a connection to gateway server
	connCtx, cancel := context.WithCancel(context.Background())
	conn, err := grpc.Dial(
		fmt.Sprintf("localhost:%s", gatewayContext.serverPort),
		grpc.WithBlock(),
		grpc.WithInsecure())
	if err != nil {
		logger.WithError(err).Errorln("failed to connect to gateway server")
		cancel()
		return
	}

	logger.WithField("state", conn.GetState().String()).Info("state of the connection to gateway server")

	eventSourceContexts[hashKey] = &EventSourceContext{
		source: &gateways.EventSource{
			Id:    hashKey,
			Name:  name,
			Value: body,
			Type:  string(gatewayContext.gateway.Spec.Type),
		},
		cancel: cancel,
		ctx:    connCtx,
		client: gateways.NewEventingClient(conn),
		conn:   conn,
	}
}

// diffConfig diffs currently active event sources and updated event sources.
// It simply matches the event source strings. So, if event source string differs through some sequence of definition
// and although the event sources are actually same, this method will treat them as different event sources.
// old event sources - event sources to be deactivate
// new event sources - new event sources to activate
func (gatewayContext *GatewayContext) diffEventSources(eventSourceContexts map[string]*EventSourceContext) ([]string, []string) {
	var staleEventSources []string
	var newEventSources []string

	var currentEventSources []string
	var updatedEventSources []string

	for currentEventSource := range gatewayContext.eventSourceContexts {
		currentEventSources = append(currentEventSources, currentEventSource)
	}
	for updatedEventSource := range eventSourceContexts {
		updatedEventSources = append(updatedEventSources, updatedEventSource)
	}

	gatewayContext.logger.WithField("current-event-sources-keys", currentEventSources).Debugln("event sources hashes")
	gatewayContext.logger.WithField("updated-event-sources-keys", updatedEventSources).Debugln("event sources hashes")

	swapped := false
	// iterates over current event sources and updated event sources
	// and creates two arrays, first one containing event sources that need to removed
	// and second containing new event sources that need to be added and run.
	for i := 0; i < 2; i++ {
		for _, currentEventSource := range currentEventSources {
			found := false
			for _, updatedEventSource := range updatedEventSources {
				if currentEventSource == updatedEventSource {
					found = true
					break
				}
			}
			if !found {
				if swapped {
					newEventSources = append(newEventSources, currentEventSource)
				} else {
					staleEventSources = append(staleEventSources, currentEventSource)
				}
			}
		}
		if i == 0 {
			currentEventSources, updatedEventSources = updatedEventSources, currentEventSources
			swapped = true
		}
	}
	return staleEventSources, newEventSources
}

// activateEventSources activate new event sources
func (gatewayContext *GatewayContext) activateEventSources(eventSources map[string]*EventSourceContext, keys []string) {
	for _, key := range keys {
		eventSource := eventSources[key]
		// register the event source
		gatewayContext.eventSourceContexts[key] = eventSource

		logger := gatewayContext.logger.WithField(common.LabelEventSource, eventSource.source.Name)

		logger.Infoln("activating new event source...")

		go func() {
			// conn should be in READY state
			if eventSource.conn.GetState() != connectivity.Ready {
				logger.Errorln("connection is not in ready state.")
				gatewayContext.statusCh <- notification{
					eventSourceNotification: &eventSourceUpdate{
						phase:   v1alpha1.NodePhaseError,
						id:      eventSource.source.Id,
						message: "connection is not in the ready state",
						name:    eventSource.source.Name,
					},
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
				gatewayContext.statusCh <- notification{
					eventSourceNotification: &eventSourceUpdate{
						phase:   v1alpha1.NodePhaseError,
						id:      eventSource.source.Id,
						message: "event_source_is_not_valid",
						name:    eventSource.source.Name,
					},
				}
				return
			}

			logger.Infoln("event source is valid")

			// mark event source as running
			gatewayContext.statusCh <- notification{
				eventSourceNotification: &eventSourceUpdate{
					phase:   v1alpha1.NodePhaseRunning,
					message: "event source is running",
					id:      eventSource.source.Id,
					name:    eventSource.source.Name,
				},
			}

			// listen to events from gateway server
			eventStream, err := eventSource.client.StartEventSource(eventSource.ctx, eventSource.source)
			if err != nil {
				logger.WithError(err).Errorln("error occurred while starting event source")
				gatewayContext.statusCh <- notification{
					eventSourceNotification: &eventSourceUpdate{
						phase:   v1alpha1.NodePhaseError,
						message: "failed to receive event stream",
						name:    eventSource.source.Name,
						id:      eventSource.source.Id,
					},
				}
				return
			}

			logger.Infoln("listening to events from gateway server...")
			for {
				event, err := eventStream.Recv()
				if err != nil {
					if err == io.EOF {
						logger.Infoln("event source has stopped")
						gatewayContext.statusCh <- notification{
							eventSourceNotification: &eventSourceUpdate{
								phase:   v1alpha1.NodePhaseCompleted,
								message: "event source has been stopped",
								name:    eventSource.source.Name,
								id:      eventSource.source.Id,
							},
						}
						return
					}

					logger.WithError(err).Errorln("failed to receive event from stream")
					gatewayContext.statusCh <- notification{
						eventSourceNotification: &eventSourceUpdate{
							phase:   v1alpha1.NodePhaseError,
							message: "failed to receive event from the event source stream",
							name:    eventSource.source.Name,
							id:      eventSource.source.Id,
						},
					}
					return
				}
				err = gatewayContext.dispatchEvent(event)
				if err != nil {
					logger.WithError(err).Errorln("failed to dispatch event to watchers")
				}
			}
		}()
	}
}

// deactivateEventSources inactivate an existing event sources
func (gatewayContext *GatewayContext) deactivateEventSources(eventSourceNames []string) {
	for _, eventSourceName := range eventSourceNames {
		eventSource := gatewayContext.eventSourceContexts[eventSourceName]
		if eventSource == nil {
			continue
		}

		logger := gatewayContext.logger.WithField(common.LabelEventSource, eventSourceName)

		logger.WithField(common.LabelEventSource, eventSource.source.Name).Infoln("stopping the event source")
		delete(gatewayContext.eventSourceContexts, eventSourceName)
		gatewayContext.statusCh <- notification{
			eventSourceNotification: &eventSourceUpdate{
				phase:   v1alpha1.NodePhaseRemove,
				id:      eventSource.source.Id,
				message: "event source is removed",
				name:    eventSource.source.Name,
			},
		}
		eventSource.cancel()
		if err := eventSource.conn.Close(); err != nil {
			logger.WithField(common.LabelEventSource, eventSource.source.Name).WithError(err).Errorln("failed to close client connection")
			continue
		}
		logger.WithField(common.LabelEventSource, eventSource.source.Name).Infoln("event source stopped")
	}
}

// syncEventSources syncs active event-sources and the updated ones
func (gatewayContext *GatewayContext) syncEventSources(eventSource *eventSourceV1Alpha1.EventSource) error {
	eventSourceContexts, err := gatewayContext.initEventSourceContexts(eventSource)
	if err != nil {
		return err
	}

	staleEventSources, newEventSources := gatewayContext.diffEventSources(eventSourceContexts)
	gatewayContext.logger.WithField(common.LabelEventSource, staleEventSources).Infoln("deleted event sources")
	gatewayContext.logger.WithField(common.LabelEventSource, newEventSources).Infoln("new event sources")

	// stop existing event sources
	gatewayContext.deactivateEventSources(staleEventSources)

	// start new event sources
	gatewayContext.activateEventSources(eventSourceContexts, newEventSources)

	gatewayContext.eventSourceContexts = eventSourceContexts

	return nil
}

// initEventSourceContext creates an internal representation of event sources.
// It returns an error if the Gateway is set in such a way
// that it wouldn't pick up any known Event Source.
func (gatewayContext *GatewayContext) initEventSourceContexts(eventSource *eventSourceV1Alpha1.EventSource) (map[string]*EventSourceContext, error) {
	eventSourceContexts := make(map[string]*EventSourceContext)
	var err error

	switch gatewayContext.gateway.Spec.Type {
	case apicommon.SNSEvent:
		for key, value := range eventSource.Spec.SNS {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.SQSEvent:
		for key, value := range eventSource.Spec.SQS {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.PubSubEvent:
		for key, value := range eventSource.Spec.PubSub {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.NATSEvent:
		for key, value := range eventSource.Spec.NATS {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.FileEvent:
		for key, value := range eventSource.Spec.File {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.CalendarEvent:
		for key, value := range eventSource.Spec.Calendar {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.AMQPEvent:
		for key, value := range eventSource.Spec.AMQP {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.GitHubEvent:
		for key, value := range eventSource.Spec.Github {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.GitLabEvent:
		for key, value := range eventSource.Spec.Gitlab {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.HDFSEvent:
		for key, value := range eventSource.Spec.HDFS {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.KafkaEvent:
		for key, value := range eventSource.Spec.Kafka {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.MinioEvent:
		for key, value := range eventSource.Spec.Minio {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.MQTTEvent:
		for key, value := range eventSource.Spec.MQTT {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.ResourceEvent:
		for key, value := range eventSource.Spec.Resource {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.SlackEvent:
		for key, value := range eventSource.Spec.Slack {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.StorageGridEvent:
		for key, value := range eventSource.Spec.StorageGrid {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.WebhookEvent:
		for key, value := range eventSource.Spec.Webhook {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.AzureEventsHub:
		for key, value := range eventSource.Spec.AzureEventsHub {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.StripeEvent:
		for key, value := range eventSource.Spec.Stripe {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.EmitterEvent:
		for key, value := range eventSource.Spec.Emitter {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.RedisEvent:
		for key, value := range eventSource.Spec.Redis {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.NSQEvent:
		for key, value := range eventSource.Spec.NSQ {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.GenericEvent:
		for key, value := range eventSource.Spec.Generic {
			gatewayContext.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	default:
		err = fmt.Errorf("gateway with type %s is invalid", gatewayContext.gateway.Spec.Type)
	}

	return eventSourceContexts, err
}
