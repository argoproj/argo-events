package eventsources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventbus"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	"github.com/argoproj/argo-events/eventsources/sources/amqp"
	"github.com/argoproj/argo-events/eventsources/sources/awssns"
	"github.com/argoproj/argo-events/eventsources/sources/awssqs"
	"github.com/argoproj/argo-events/eventsources/sources/azureeventshub"
	"github.com/argoproj/argo-events/eventsources/sources/calendar"
	"github.com/argoproj/argo-events/eventsources/sources/emitter"
	"github.com/argoproj/argo-events/eventsources/sources/file"
	"github.com/argoproj/argo-events/eventsources/sources/gcppubsub"
	"github.com/argoproj/argo-events/eventsources/sources/github"
	"github.com/argoproj/argo-events/eventsources/sources/gitlab"
	"github.com/argoproj/argo-events/eventsources/sources/hdfs"
	"github.com/argoproj/argo-events/eventsources/sources/kafka"
	"github.com/argoproj/argo-events/eventsources/sources/minio"
	"github.com/argoproj/argo-events/eventsources/sources/mqtt"
	"github.com/argoproj/argo-events/eventsources/sources/nats"
	"github.com/argoproj/argo-events/eventsources/sources/nsq"
	"github.com/argoproj/argo-events/eventsources/sources/redis"
	"github.com/argoproj/argo-events/eventsources/sources/resource"
	"github.com/argoproj/argo-events/eventsources/sources/slack"
	"github.com/argoproj/argo-events/eventsources/sources/storagegrid"
	"github.com/argoproj/argo-events/eventsources/sources/stripe"
	"github.com/argoproj/argo-events/eventsources/sources/webhook"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

// EventingServer is the server API for Eventing service.
type EventingServer interface {

	// ValidateEventSource validates an event source.
	ValidateEventSource(context.Context) error

	GetEventSourceName() string

	GetEventName() string

	GetEventSourceType() apicommon.EventSourceType

	// Function to start listening events.
	StartListening(ctx context.Context, dispatch func([]byte) error) error
}

// GetEventingServers returns the mapping of event source type and list of eventing servers
func GetEventingServers(eventSource *v1alpha1.EventSource) map[apicommon.EventSourceType][]EventingServer {
	result := make(map[apicommon.EventSourceType][]EventingServer)
	if len(eventSource.Spec.AMQP) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.AMQP {
			servers = append(servers, &amqp.EventListener{EventSourceName: eventSource.Name, EventName: k, AMQPEventSource: v})
		}
		result[apicommon.AMQPEvent] = servers
	}
	if len(eventSource.Spec.AzureEventsHub) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.AzureEventsHub {
			servers = append(servers, &azureeventshub.EventListener{EventSourceName: eventSource.Name, EventName: k, AzureEventsHubEventSource: v})
		}
		result[apicommon.AzureEventsHub] = servers
	}
	if len(eventSource.Spec.Calendar) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Calendar {
			servers = append(servers, &calendar.EventListener{EventSourceName: eventSource.Name, EventName: k, CalendarEventSource: v})
		}
		result[apicommon.CalendarEvent] = servers
	}
	if len(eventSource.Spec.Emitter) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Emitter {
			servers = append(servers, &emitter.EventListener{EventSourceName: eventSource.Name, EventName: k, EmitterEventSource: v})
		}
		result[apicommon.EmitterEvent] = servers
	}
	if len(eventSource.Spec.File) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.File {
			servers = append(servers, &file.EventListener{EventSourceName: eventSource.Name, EventName: k, FileEventSource: v})
		}
		result[apicommon.FileEvent] = servers
	}
	if len(eventSource.Spec.Github) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Github {
			servers = append(servers, &github.EventListener{EventSourceName: eventSource.Name, EventName: k, GithubEventSource: v})
		}
		result[apicommon.GithubEvent] = servers
	}
	if len(eventSource.Spec.Gitlab) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Gitlab {
			servers = append(servers, &gitlab.EventListener{EventSourceName: eventSource.Name, EventName: k, GitlabEventSource: v})
		}
		result[apicommon.GitlabEvent] = servers
	}
	if len(eventSource.Spec.HDFS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.HDFS {
			servers = append(servers, &hdfs.EventListener{EventSourceName: eventSource.Name, EventName: k, HDFSEventSource: v})
		}
		result[apicommon.HDFSEvent] = servers
	}
	if len(eventSource.Spec.Kafka) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Kafka {
			servers = append(servers, &kafka.EventListener{EventSourceName: eventSource.Name, EventName: k, KafkaEventSource: v})
		}
		result[apicommon.KafkaEvent] = servers
	}
	if len(eventSource.Spec.MQTT) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.MQTT {
			servers = append(servers, &mqtt.EventListener{EventSourceName: eventSource.Name, EventName: k, MQTTEventSource: v})
		}
		result[apicommon.MQTTEvent] = servers
	}
	if len(eventSource.Spec.Minio) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Minio {
			servers = append(servers, &minio.EventListener{EventSourceName: eventSource.Name, EventName: k, MinioEventSource: v})
		}
		result[apicommon.MinioEvent] = servers
	}
	if len(eventSource.Spec.NATS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.NATS {
			servers = append(servers, &nats.EventListener{EventSourceName: eventSource.Name, EventName: k, NATSEventSource: v})
		}
		result[apicommon.NATSEvent] = servers
	}
	if len(eventSource.Spec.NSQ) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.NSQ {
			servers = append(servers, &nsq.EventListener{EventSourceName: eventSource.Name, EventName: k, NSQEventSource: v})
		}
		result[apicommon.NSQEvent] = servers
	}
	if len(eventSource.Spec.PubSub) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.PubSub {
			servers = append(servers, &gcppubsub.EventListener{EventSourceName: eventSource.Name, EventName: k, PubSubEventSource: v})
		}
		result[apicommon.PubSubEvent] = servers
	}
	if len(eventSource.Spec.Redis) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Redis {
			servers = append(servers, &redis.EventListener{EventSourceName: eventSource.Name, EventName: k, RedisEventSource: v})
		}
		result[apicommon.RedisEvent] = servers
	}
	if len(eventSource.Spec.SNS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.SNS {
			servers = append(servers, &awssns.EventListener{EventSourceName: eventSource.Name, EventName: k, SNSEventSource: v})
		}
		result[apicommon.SNSEvent] = servers
	}
	if len(eventSource.Spec.SQS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.SQS {
			servers = append(servers, &awssqs.EventListener{EventSourceName: eventSource.Name, EventName: k, SQSEventSource: v})
		}
		result[apicommon.SQSEvent] = servers
	}
	if len(eventSource.Spec.Slack) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Slack {
			servers = append(servers, &slack.EventListener{EventSourceName: eventSource.Name, EventName: k, SlackEventSource: v})
		}
		result[apicommon.SlackEvent] = servers
	}
	if len(eventSource.Spec.StorageGrid) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.StorageGrid {
			servers = append(servers, &storagegrid.EventListener{EventSourceName: eventSource.Name, EventName: k, StorageGridEventSource: v})
		}
		result[apicommon.StorageGridEvent] = servers
	}
	if len(eventSource.Spec.Stripe) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Stripe {
			servers = append(servers, &stripe.EventListener{EventSourceName: eventSource.Name, EventName: k, StripeEventSource: v})
		}
		result[apicommon.StripeEvent] = servers
	}
	if len(eventSource.Spec.Webhook) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Webhook {
			servers = append(servers, &webhook.EventListener{EventSourceName: eventSource.Name, EventName: k, WebhookContext: v})
		}
		result[apicommon.WebhookEvent] = servers
	}
	if len(eventSource.Spec.Resource) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Resource {
			servers = append(servers, &resource.EventListener{EventSourceName: eventSource.Name, EventName: k, ResourceEventSource: v})
		}
		result[apicommon.ResourceEvent] = servers
	}
	return result
}

// EventSourceAdaptor is the adaptor for eventsource service
type EventSourceAdaptor struct {
	eventSource     *v1alpha1.EventSource
	eventBusConfig  *eventbusv1alpha1.BusConfig
	eventBusSubject string
	hostname        string

	eventBusConn eventbusdriver.Connection
}

// NewEventSourceAdaptor returns a new EventSourceAdaptor
func NewEventSourceAdaptor(eventSource *v1alpha1.EventSource, eventBusConfig *eventbusv1alpha1.BusConfig, eventBusSubject, hostname string) *EventSourceAdaptor {
	return &EventSourceAdaptor{
		eventSource:     eventSource,
		eventBusConfig:  eventBusConfig,
		eventBusSubject: eventBusSubject,
		hostname:        hostname,
	}
}

// Start function
func (e *EventSourceAdaptor) Start(ctx context.Context, stopCh <-chan struct{}) error {
	logger := logging.FromContext(ctx)
	logger.Info("Starting event source server...")
	servers := GetEventingServers(e.eventSource)
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	driver, err := eventbus.GetDriver(cctx, *e.eventBusConfig, e.eventBusSubject, e.hostname)
	if err != nil {
		logger.WithError(err).Errorln("failed to get eventbus driver")
		return err
	}
	e.eventBusConn, err = driver.Connect()
	if err != nil {
		logger.WithError(err).Errorln("failed to connect to eventbus")
		return err
	}
	defer e.eventBusConn.Close()

	// Daemon to reconnect
	go func() {
		logger.Info("starting eventbus connection daemon...")
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-cctx.Done():
				logger.Info("exiting eventbus connection daemon...")
				return
			case <-ticker.C:
				if e.eventBusConn == nil || e.eventBusConn.IsClosed() {
					logger.Info("NATS connection lost, reconnecting...")
					e.eventBusConn, err = driver.Connect()
					if err != nil {
						logger.WithError(err).Errorln("failed to reconnect to eventbus")
						continue
					}
					logger.Info("reconnected the NATS streaming server...")
				}
			}
		}
	}()

	for _, ss := range servers {
		for _, server := range ss {
			err := server.ValidateEventSource(cctx)
			if err != nil {
				logger.WithError(err).Errorln("Validation failed.")
				return err
			}
			go func(s EventingServer) {
				err := s.StartListening(cctx, func(data []byte) error {
					event := cloudevents.NewEvent()
					event.SetID(fmt.Sprintf("%x", uuid.New()))
					event.SetType(string(s.GetEventSourceType()))
					event.SetSource(s.GetEventSourceName())
					event.SetSubject(s.GetEventName())
					event.SetTime(time.Now())
					err := event.SetData(cloudevents.ApplicationJSON, data)
					if err != nil {
						return err
					}
					eventBody, err := json.Marshal(event)
					if err != nil {
						return err
					}
					if e.eventBusConn == nil || e.eventBusConn.IsClosed() {
						return errors.New("failed to publish event, eventbus connection closed.")
					}
					return driver.Publish(e.eventBusConn, eventBody)
				})
				logger.WithField(logging.LabelEventSourceName, s.GetEventSourceName()).
					WithField(logging.LabelEventName, s.GetEventName()).WithError(err).Errorln("failed to start service.")
			}(server)
		}
	}
	logger.Info("Eventing server started.")
	<-stopCh
	logger.Info("Shutting down...")
	return nil
}
