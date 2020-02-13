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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	eventSourceV1Alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"io"
)

// populateEventSourceContexts sets up the contexts for event sources
func (gc *GatewayContext) populateEventSourceContexts(name string, value interface{}, eventSourceContexts map[string]*EventSourceContext) {
	body, err := yaml.Marshal(value)
	if err != nil {
		gc.logger.WithField("event-source-name", name).Errorln("failed to marshal the event source value, won't process it")
		return
	}

	fmt.Printf("%s\n", string(body))

	hashKey := common.Hasher(name + string(body))

	logger := gc.logger.WithFields(logrus.Fields{
		"name":  name,
		"value": value,
	})

	logger.WithField("hash", hashKey).Debugln("hash of the event source")

	// create a connection to gateway server
	connCtx, cancel := context.WithCancel(context.Background())
	conn, err := grpc.Dial(
		fmt.Sprintf("localhost:%s", gc.serverPort),
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
			Type:  string(gc.gateway.Spec.Type),
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
func (gc *GatewayContext) diffEventSources(eventSourceContexts map[string]*EventSourceContext) (staleEventSources []string, newEventSources []string) {
	var currentEventSources []string
	var updatedEventSources []string

	for currentEventSource := range gc.eventSourceContexts {
		currentEventSources = append(currentEventSources, currentEventSource)
	}
	for updatedEventSource := range eventSourceContexts {
		updatedEventSources = append(updatedEventSources, updatedEventSource)
	}

	gc.logger.WithField("current-event-sources-keys", currentEventSources).Debugln("event sources hashes")
	gc.logger.WithField("updated-event-sources-keys", updatedEventSources).Debugln("event sources hashes")

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
	return
}

// newConn returns a new gRPC connection to the gateway server.
func (gc *GatewayContext) newConn() (*grpc.ClientConn, error) {
	return grpc.Dial(
		fmt.Sprintf("localhost:%s", gc.serverPort),
		grpc.WithBlock(),
		grpc.WithInsecure())
}

func (gc *GatewayContext) operateEventSource(esctx *EventSourceContext) {
	logger := gc.logger.WithField(common.LabelEventSource, esctx.source.Name)

	logger.Infoln("operating on the event source...")

	logger.Infoln("setting up a new connection to the gateway server")
	conn, err := gc.newConn()
	if err != nil {
		logger.WithError(err).Errorln("failed to connect with the gateway server")
		gc.statusCh <- notification{
			eventSourceNotification: &eventSourceUpdate{
				phase:   v1alpha1.NodePhaseError,
				id:      esctx.source.Id,
				message: fmt.Sprintf("failed to connect to the gateway server. err: %+v", err),
				name:    esctx.source.Name,
			},
		}
		return
	}

	defer conn.Close()

	// conn should be in READY state
	if esctx.conn.GetState() != connectivity.Ready {
		logger.Errorln("connection is not in ready state.")
		gc.statusCh <- notification{
			eventSourceNotification: &eventSourceUpdate{
				phase:   v1alpha1.NodePhaseError,
				id:      esctx.source.Id,
				message: "connection is not in the ready state",
				name:    esctx.source.Name,
			},
		}
		return
	}

	client := gateways.NewEventingClient(conn)

	ctx, cancel := context.WithCancel(context.Background())

	// validate event source
	if valid, _ := client.ValidateEventSource(ctx, esctx.source); !valid.IsValid {
		logger.WithField("reason", valid.Reason).Errorln("event source is not valid")
		gc.statusCh <- notification{
			eventSourceNotification: &eventSourceUpdate{
				phase:   v1alpha1.NodePhaseError,
				id:      esctx.source.Id,
				message: fmt.Sprintf("event source is not valid. reason: %s", valid.Reason),
				name:    esctx.source.Name,
			},
		}
		return
	}

	logger.Infoln("event source is valid")

	// mark event source as running
	gc.statusCh <- notification{
		eventSourceNotification: &eventSourceUpdate{
			phase:   v1alpha1.NodePhaseRunning,
			message: "event source is running",
			id:      esctx.source.Id,
			name:    esctx.source.Name,
		},
	}

	// listen to events from gateway server
	eventStream, err := client.StartEventSource(ctx, esctx.source)
	if err != nil {
		logger.WithError(err).Errorln("error occurred while starting event source")
		gc.statusCh <- notification{
			eventSourceNotification: &eventSourceUpdate{
				phase:   v1alpha1.NodePhaseError,
				message: "failed to receive event stream",
				name:    esctx.source.Name,
				id:      esctx.source.Id,
			},
		}
		return
	}

	for {
		select {
		case <-esctx.stop:

		default:
		event, err := eventStream.Recv()
			if err != nil {
				if err == io.EOF {
					logger.Infoln("event source has stopped")
					gc.statusCh <- notification{
						eventSourceNotification: &eventSourceUpdate{
							phase:   v1alpha1.NodePhaseCompleted,
							message: "event source has been stopped",
							name:    esctx.source.Name,
							id:      esctx.source.Id,
						},
					}
					return
				}

				logger.WithError(err).Errorln("failed to receive event from stream")
				gc.statusCh <- notification{
					eventSourceNotification: &eventSourceUpdate{
						phase:   v1alpha1.NodePhaseError,
						message: "failed to receive event from the event source stream",
						name:    esctx.source.Name,
						id:      esctx.source.Id,
					},
				}
				eventStream.CloseSend()
				return
			}
			err = gc.dispatchEvent(event)
			if err != nil {
				logger.WithError(err).Errorln("failed to dispatch event to watchers")
			}
		}
	}

	logger.Infoln("listening to events from gateway server...")
}

// activateEventSources activate new event sources
func (gc *GatewayContext) activateEventSources(eventSources map[string]*EventSourceContext, keys []string) {
	for _, key := range keys {
		eventSource := eventSources[key]
		// register the event source
		gc.eventSourceContexts[key] = eventSource

		logger := gc.logger.WithField(common.LabelEventSource, eventSource.source.Name)

		logger.Infoln("activating new event source...")

		go func() {
			// conn should be in READY state
			if eventSource.conn.GetState() != connectivity.Ready {
				logger.Errorln("connection is not in ready state.")
				gc.statusCh <- notification{
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
				gc.statusCh <- notification{
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
			gc.statusCh <- notification{
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
				gc.statusCh <- notification{
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
						gc.statusCh <- notification{
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
					gc.statusCh <- notification{
						eventSourceNotification: &eventSourceUpdate{
							phase:   v1alpha1.NodePhaseError,
							message: "failed to receive event from the event source stream",
							name:    eventSource.source.Name,
							id:      eventSource.source.Id,
						},
					}
					eventStream.CloseSend()
					return
				}
				err = gc.dispatchEvent(event)
				if err != nil {
					logger.WithError(err).Errorln("failed to dispatch event to watchers")
				}
			}
		}()
	}
}

// deactivateEventSources inactivate an existing event sources
func (gc *GatewayContext) deactivateEventSources(eventSourceNames []string) {
	for _, eventSourceName := range eventSourceNames {
		eventSource := gc.eventSourceContexts[eventSourceName]
		if eventSource == nil {
			continue
		}

		logger := gc.logger.WithField(common.LabelEventSource, eventSourceName)

		logger.WithField(common.LabelEventSource, eventSource.source.Name).Infoln("stopping the event source")
		delete(gc.eventSourceContexts, eventSourceName)
		gc.statusCh <- notification{
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
		}
	}
}

// getOutOfSyncEventSources returns event sources that are stale and out of sync.
func (gc *GatewayContext) getOutOfSyncEventSources(eventSource *eventSourceV1Alpha1.EventSource) []string {
	var result []string
	for name, source := range
}

// syncEventSources syncs active event-sources and the updated ones
func (gc *GatewayContext) syncEventSources(eventSource *eventSourceV1Alpha1.EventSource) error {
	eventSourceContexts := gc.initEventSourceContexts(eventSource)

	staleEventSources, newEventSources := gc.diffEventSources(eventSourceContexts)
	gc.logger.WithField(common.LabelEventSource, staleEventSources).Infoln("deleted event sources")
	gc.logger.WithField(common.LabelEventSource, newEventSources).Infoln("new event sources")

	// stop existing event sources
	gc.deactivateEventSources(staleEventSources)

	// start new event sources
	gc.activateEventSources(eventSourceContexts, newEventSources)

	return nil
}

// initEventSourceContext creates an internal representation of event sources.
func (gc *GatewayContext) initEventSourceContexts(eventSource *eventSourceV1Alpha1.EventSource) map[string]*EventSourceContext {
	eventSourceContexts := make(map[string]*EventSourceContext)

	switch gc.gateway.Spec.Type {
	case apicommon.SNSEvent:
		for key, value := range eventSource.Spec.SNS {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.SQSEvent:
		for key, value := range eventSource.Spec.SQS {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.PubSubEvent:
		for key, value := range eventSource.Spec.PubSub {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.NATSEvent:
		for key, value := range eventSource.Spec.NATS {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.FileEvent:
		for key, value := range eventSource.Spec.File {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.CalendarEvent:
		for key, value := range eventSource.Spec.Calendar {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.AMQPEvent:
		for key, value := range eventSource.Spec.AMQP {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.GitHubEvent:
		for key, value := range eventSource.Spec.Github {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.GitLabEvent:
		for key, value := range eventSource.Spec.Gitlab {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.HDFSEvent:
		for key, value := range eventSource.Spec.HDFS {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.KafkaEvent:
		for key, value := range eventSource.Spec.Kafka {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.MinioEvent:
		for key, value := range eventSource.Spec.Minio {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.MQTTEvent:
		for key, value := range eventSource.Spec.MQTT {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.ResourceEvent:
		for key, value := range eventSource.Spec.Resource {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.SlackEvent:
		for key, value := range eventSource.Spec.Slack {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.StorageGridEvent:
		for key, value := range eventSource.Spec.StorageGrid {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.WebhookEvent:
		for key, value := range eventSource.Spec.Webhook {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.AzureEventsHub:
		for key, value := range eventSource.Spec.AzureEventsHub {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.StripeEvent:
		for key, value := range eventSource.Spec.Stripe {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.EmitterEvent:
		for key, value := range eventSource.Spec.Emitter {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.RedisEvent:
		for key, value := range eventSource.Spec.Redis {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.NSQEvent:
		for key, value := range eventSource.Spec.NSQ {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	case apicommon.GenericEvent:
		for key, value := range eventSource.Spec.Generic {
			gc.populateEventSourceContexts(key, value, eventSourceContexts)
		}
	}

	return eventSourceContexts
}
