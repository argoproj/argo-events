package eventsources

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventbus"
	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/alibabacloudmns"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/amqp"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/awssns"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/awssqs"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/azureeventshub"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/azurequeuestorage"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/azureservicebus"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/bitbucket"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/bitbucketserver"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/calendar"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/emitter"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/file"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/gcppubsub"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/generic"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/gerrit"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/github"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/gitlab"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/hdfs"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/kafka"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/minio"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/mqtt"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/nats"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/nsq"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/pulsar"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/redis"
	redisstream "github.com/argoproj/argo-events/pkg/eventsources/sources/redis_stream"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/resource"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/sftp"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/slack"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/storagegrid"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/stripe"
	"github.com/argoproj/argo-events/pkg/eventsources/sources/webhook"
	eventsourcemetrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/expr"
	"github.com/argoproj/argo-events/pkg/shared/leaderelection"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// EventingServer is the server API for Eventing service.
type EventingServer interface {

	// ValidateEventSource validates an event source.
	ValidateEventSource(context.Context) error

	GetEventSourceName() string

	GetEventName() string

	GetEventSourceType() aev1.EventSourceType

	// Function to start listening events.
	StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error
}

// GetEventingServers returns the mapping of event source type and list of eventing servers
func GetEventingServers(eventSource *aev1.EventSource, metrics *eventsourcemetrics.Metrics) (map[aev1.EventSourceType][]EventingServer, map[string]*aev1.EventSourceFilter) {
	result := make(map[aev1.EventSourceType][]EventingServer)
	filters := make(map[string]*aev1.EventSourceFilter)
	if len(eventSource.Spec.AMQP) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.AMQP {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &amqp.EventListener{EventSourceName: eventSource.Name, EventName: k, AMQPEventSource: v, Metrics: metrics})
		}
		result[aev1.AMQPEvent] = servers
	}
	if len(eventSource.Spec.AzureEventsHub) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.AzureEventsHub {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &azureeventshub.EventListener{EventSourceName: eventSource.Name, EventName: k, AzureEventsHubEventSource: v, Metrics: metrics})
		}
		result[aev1.AzureEventsHub] = servers
	}
	if len(eventSource.Spec.AzureQueueStorage) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.AzureQueueStorage {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &azurequeuestorage.EventListener{EventSourceName: eventSource.Name, EventName: k, AzureQueueStorageEventSource: v, Metrics: metrics})
		}
		result[aev1.AzureQueueStorage] = servers
	}
	if len(eventSource.Spec.AzureServiceBus) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.AzureServiceBus {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &azureservicebus.EventListener{EventSourceName: eventSource.Name, EventName: k, AzureServiceBusEventSource: v, Metrics: metrics})
		}
		result[aev1.AzureServiceBus] = servers
	}
	if len(eventSource.Spec.Bitbucket) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Bitbucket {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &bitbucket.EventListener{EventSourceName: eventSource.Name, EventName: k, BitbucketEventSource: v, Metrics: metrics})
		}
		result[aev1.BitbucketEvent] = servers
	}
	if len(eventSource.Spec.BitbucketServer) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.BitbucketServer {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &bitbucketserver.EventListener{EventSourceName: eventSource.Name, EventName: k, BitbucketServerEventSource: v, Metrics: metrics})
		}
		result[aev1.BitbucketServerEvent] = servers
	}
	if len(eventSource.Spec.Calendar) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Calendar {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &calendar.EventListener{EventSourceName: eventSource.Name, EventName: k, CalendarEventSource: v, Namespace: eventSource.Namespace, Metrics: metrics})
		}
		result[aev1.CalendarEvent] = servers
	}
	if len(eventSource.Spec.Emitter) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Emitter {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &emitter.EventListener{EventSourceName: eventSource.Name, EventName: k, EmitterEventSource: v, Metrics: metrics})
		}
		result[aev1.EmitterEvent] = servers
	}
	if len(eventSource.Spec.File) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.File {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &file.EventListener{EventSourceName: eventSource.Name, EventName: k, FileEventSource: v, Metrics: metrics})
		}
		result[aev1.FileEvent] = servers
	}
	if len(eventSource.Spec.SFTP) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.SFTP {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &sftp.EventListener{EventSourceName: eventSource.Name, EventName: k, SFTPEventSource: v, Metrics: metrics})
		}
		result[aev1.SFTPEvent] = servers
	}
	if len(eventSource.Spec.Gerrit) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Gerrit {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &gerrit.EventListener{EventSourceName: eventSource.Name, EventName: k, GerritEventSource: v, Metrics: metrics})
		}
		result[aev1.GerritEvent] = servers
	}
	if len(eventSource.Spec.Github) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Github {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &github.EventListener{EventSourceName: eventSource.Name, EventName: k, GithubEventSource: v, Metrics: metrics})
		}
		result[aev1.GithubEvent] = servers
	}
	if len(eventSource.Spec.Gitlab) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Gitlab {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &gitlab.EventListener{EventSourceName: eventSource.Name, EventName: k, GitlabEventSource: v, Metrics: metrics})
		}
		result[aev1.GitlabEvent] = servers
	}
	if len(eventSource.Spec.HDFS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.HDFS {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &hdfs.EventListener{EventSourceName: eventSource.Name, EventName: k, HDFSEventSource: v, Metrics: metrics})
		}
		result[aev1.HDFSEvent] = servers
	}
	if len(eventSource.Spec.Kafka) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Kafka {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &kafka.EventListener{EventSourceName: eventSource.Name, EventName: k, KafkaEventSource: v, Metrics: metrics})
		}
		result[aev1.KafkaEvent] = servers
	}
	if len(eventSource.Spec.MQTT) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.MQTT {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &mqtt.EventListener{EventSourceName: eventSource.Name, EventName: k, MQTTEventSource: v, Metrics: metrics})
		}
		result[aev1.MQTTEvent] = servers
	}
	if len(eventSource.Spec.Minio) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Minio {
			servers = append(servers, &minio.EventListener{EventSourceName: eventSource.Name, EventName: k, MinioEventSource: v, Metrics: metrics})
		}
		result[aev1.MinioEvent] = servers
	}
	if len(eventSource.Spec.NATS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.NATS {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &nats.EventListener{EventSourceName: eventSource.Name, EventName: k, NATSEventSource: v, Metrics: metrics})
		}
		result[aev1.NATSEvent] = servers
	}
	if len(eventSource.Spec.NSQ) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.NSQ {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &nsq.EventListener{EventSourceName: eventSource.Name, EventName: k, NSQEventSource: v, Metrics: metrics})
		}
		result[aev1.NSQEvent] = servers
	}
	if len(eventSource.Spec.PubSub) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.PubSub {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &gcppubsub.EventListener{EventSourceName: eventSource.Name, EventName: k, PubSubEventSource: v, Metrics: metrics})
		}
		result[aev1.PubSubEvent] = servers
	}
	if len(eventSource.Spec.Redis) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Redis {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &redis.EventListener{EventSourceName: eventSource.Name, EventName: k, RedisEventSource: v, Metrics: metrics})
		}
		result[aev1.RedisEvent] = servers
	}
	if len(eventSource.Spec.RedisStream) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.RedisStream {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &redisstream.EventListener{EventSourceName: eventSource.Name, EventName: k, EventSource: v, Metrics: metrics})
		}
		result[aev1.RedisStreamEvent] = servers
	}
	if len(eventSource.Spec.SNS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.SNS {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &awssns.EventListener{EventSourceName: eventSource.Name, EventName: k, SNSEventSource: v, Metrics: metrics})
		}
		result[aev1.SNSEvent] = servers
	}
	if len(eventSource.Spec.SQS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.SQS {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &awssqs.EventListener{EventSourceName: eventSource.Name, EventName: k, SQSEventSource: v, Metrics: metrics})
		}
		result[aev1.SQSEvent] = servers
	}
	if len(eventSource.Spec.Slack) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Slack {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &slack.EventListener{EventSourceName: eventSource.Name, EventName: k, SlackEventSource: v, Metrics: metrics})
		}
		result[aev1.SlackEvent] = servers
	}
	if len(eventSource.Spec.StorageGrid) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.StorageGrid {
			servers = append(servers, &storagegrid.EventListener{EventSourceName: eventSource.Name, EventName: k, StorageGridEventSource: v, Metrics: metrics})
		}
		result[aev1.StorageGridEvent] = servers
	}
	if len(eventSource.Spec.Stripe) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Stripe {
			servers = append(servers, &stripe.EventListener{EventSourceName: eventSource.Name, EventName: k, StripeEventSource: v, Metrics: metrics})
		}
		result[aev1.StripeEvent] = servers
	}
	if len(eventSource.Spec.Webhook) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Webhook {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &webhook.EventListener{EventSourceName: eventSource.Name, EventName: k, Webhook: v, Metrics: metrics})
		}
		result[aev1.WebhookEvent] = servers
	}
	if len(eventSource.Spec.Resource) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Resource {
			servers = append(servers, &resource.EventListener{EventSourceName: eventSource.Name, EventName: k, ResourceEventSource: v, Metrics: metrics})
		}
		result[aev1.ResourceEvent] = servers
	}
	if len(eventSource.Spec.Pulsar) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Pulsar {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &pulsar.EventListener{EventSourceName: eventSource.Name, EventName: k, PulsarEventSource: v, Metrics: metrics})
		}
		result[aev1.PulsarEvent] = servers
	}
	if len(eventSource.Spec.Generic) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Generic {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &generic.EventListener{EventSourceName: eventSource.Name, EventName: k, GenericEventSource: v, Metrics: metrics})
		}
		result[aev1.GenericEvent] = servers
	}
	if len(eventSource.Spec.MNS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.MNS {
			if v.Filter != nil {
				filters[k] = v.Filter
			}
			servers = append(servers, &alibabacloudmns.EventListener{EventSourceName: eventSource.Name, EventName: k, MNSEventSource: v, Metrics: metrics})
		}
		result[aev1.MNSEvent] = servers
	}
	return result, filters
}

// EventSourceAdaptor is the adaptor for eventsource service
type EventSourceAdaptor struct {
	eventSource     *aev1.EventSource
	eventBusConfig  *aev1.BusConfig
	eventBusSubject string
	hostname        string

	eventBusConn eventbuscommon.EventSourceConnection

	metrics *eventsourcemetrics.Metrics
}

// NewEventSourceAdaptor returns a new EventSourceAdaptor
func NewEventSourceAdaptor(eventSource *aev1.EventSource, eventBusConfig *aev1.BusConfig, eventBusSubject, hostname string, metrics *eventsourcemetrics.Metrics) *EventSourceAdaptor {
	return &EventSourceAdaptor{
		eventSource:     eventSource,
		eventBusConfig:  eventBusConfig,
		eventBusSubject: eventBusSubject,
		hostname:        hostname,
		metrics:         metrics,
	}
}

// Start function
func (e *EventSourceAdaptor) Start(ctx context.Context) error {
	log := logging.FromContext(ctx)

	recreateTypes := make(map[aev1.EventSourceType]bool)
	for _, esType := range aev1.RecreateStrategyEventSources {
		recreateTypes[esType] = true
	}
	isRecreateType := false
	servers, filters := GetEventingServers(e.eventSource, e.metrics)
	for k := range servers {
		if _, ok := recreateTypes[k]; ok {
			isRecreateType = true
		}
		// This is based on the presumption that all the events in one
		// EventSource object use the same type of deployment strategy
		break
	}

	if !isRecreateType {
		return e.run(ctx, servers, filters)
	}

	clusterName := fmt.Sprintf("%s-eventsource-%s", e.eventSource.Namespace, e.eventSource.Name)
	replicas := int(e.eventSource.Spec.GetReplicas())
	leasename := fmt.Sprintf("eventsource-%s", e.eventSource.Name)

	elector, err := leaderelection.NewElector(ctx, *e.eventBusConfig, clusterName, replicas, e.eventSource.Namespace, leasename, e.hostname)
	if err != nil {
		log.Errorw("failed to get an elector", zap.Error(err))
		return err
	}

	elector.RunOrDie(ctx, leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			if err := e.run(ctx, servers, filters); err != nil {
				log.Fatalw("failed to start", zap.Error(err))
			}
		},
		OnStoppedLeading: func() {
			log.Fatalf("leader lost: %s", e.hostname)
		},
	})

	return nil
}

func (e *EventSourceAdaptor) run(ctx context.Context, servers map[aev1.EventSourceType][]EventingServer, filters map[string]*aev1.EventSourceFilter) error {
	logger := logging.FromContext(ctx)
	logger.Info("Starting event source server...")
	clientID := generateClientID(e.hostname)
	driver, err := eventbus.GetEventSourceDriver(ctx, *e.eventBusConfig, e.eventSource.Name, e.eventBusSubject)
	if err != nil {
		logger.Errorw("failed to get eventbus driver", zap.Error(err))
		return err
	}
	if err = sharedutil.DoWithRetry(&sharedutil.DefaultBackoff, func() error {
		err = driver.Initialize()
		if err != nil {
			return err
		}
		e.eventBusConn, err = driver.Connect(clientID)
		return err
	}); err != nil {
		logger.Errorw("failed to connect to eventbus", zap.Error(err))
		return err
	}

	defer e.eventBusConn.Close()

	ctx, cancel := context.WithCancel(ctx)
	connWG := &sync.WaitGroup{}

	// Daemon to reconnect
	connWG.Add(1)
	go func() {
		defer connWG.Done()
		logger.Info("starting eventbus connection daemon...")
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info("exiting eventbus connection daemon...")
				return
			case <-ticker.C:
				if e.eventBusConn == nil || e.eventBusConn.IsClosed() {
					logger.Info("NATS connection lost, reconnecting...")
					// Regenerate the client ID to avoid the issue that NAT server still thinks the client is alive.
					clientID := generateClientID(e.hostname)
					driver, err := eventbus.GetEventSourceDriver(ctx, *e.eventBusConfig, e.eventSource.Name, e.eventBusSubject)
					if err != nil {
						logger.Errorw("failed to get eventbus driver during reconnection", zap.Error(err))
						continue
					}
					e.eventBusConn, err = driver.Connect(clientID)
					if err != nil {
						logger.Errorw("failed to reconnect to eventbus", zap.Error(err))
						continue
					}
					logger.Info("reconnected to eventbus successfully")
				}
			}
		}
	}()

	wg := &sync.WaitGroup{}
	for _, ss := range servers {
		for _, server := range ss {
			// Validation has been done in eventsource-controller, it's harmless to do it again here.
			err := server.ValidateEventSource(ctx)
			if err != nil {
				logger.Errorw("Validation failed", zap.Error(err), zap.Any(logging.LabelEventName,
					server.GetEventName()), zap.Any(logging.LabelEventSourceType, server.GetEventSourceType()))
				// Continue starting other event services instead of failing all of them
				continue
			}

			e.metrics.InitEventMetrics(server.GetEventSourceName(), server.GetEventName())

			wg.Add(1)
			go func(s EventingServer) {
				defer wg.Done()
				e.metrics.IncRunningServices(s.GetEventSourceName())
				defer e.metrics.DecRunningServices(s.GetEventSourceName())
				duration := aev1.FromString("1s")
				factor := aev1.NewAmount("1")
				jitter := aev1.NewAmount("30")
				backoff := aev1.Backoff{
					Steps:    10,
					Duration: &duration,
					Factor:   &factor,
					Jitter:   &jitter,
				}
				if err = sharedutil.DoWithRetry(&backoff, func() error {
					return s.StartListening(ctx, func(data []byte, opts ...eventsourcecommon.Option) error {
						if filter, ok := filters[s.GetEventName()]; ok {
							proceed, err := filterEvent(data, filter)
							if err != nil {
								logger.Errorw("Failed to filter event", zap.Error(err))
								return nil
							}
							if !proceed {
								logger.Debug("Filter condition not met, skip dispatching")
								return nil
							}
						}

						uuidNew := uuid.New()
						event := cloudevents.NewEvent()
						event.SetID(fmt.Sprintf("%x", uuidNew[:]))
						event.SetType(string(s.GetEventSourceType()))
						event.SetSource(s.GetEventSourceName())
						event.SetSubject(s.GetEventName())
						event.SetTime(time.Now())
						for _, opt := range opts {
							err := opt(&event)
							if err != nil {
								return err
							}
						}
						err := event.SetData(cloudevents.ApplicationJSON, data)
						if err != nil {
							return err
						}
						eventBody, err := json.Marshal(event)
						if err != nil {
							return err
						}

						if e.eventBusConn == nil || e.eventBusConn.IsClosed() {
							return eventbuscommon.NewEventBusError(fmt.Errorf("failed to publish event, eventbus connection closed"))
						}

						msg := eventbuscommon.Message{
							MsgHeader: eventbuscommon.MsgHeader{
								EventSourceName: s.GetEventSourceName(),
								EventName:       s.GetEventName(),
								ID:              event.ID(),
							},
							Body: eventBody,
						}
						logger.Debugw(string(data), zap.String("eventID", event.ID()))
						if err = sharedutil.DoWithRetry(&sharedutil.DefaultBackoff, func() error {
							return e.eventBusConn.Publish(ctx, msg)
						}); err != nil {
							logger.Errorw("Failed to publish an event", zap.Error(err), zap.String(logging.LabelEventName,
								s.GetEventName()), zap.Any(logging.LabelEventSourceType, s.GetEventSourceType()), zap.String("eventID", event.ID()))
							e.metrics.EventSentFailed(s.GetEventSourceName(), s.GetEventName())
							return eventbuscommon.NewEventBusError(err)
						}
						logger.Infow("Succeeded to publish an event", zap.String(logging.LabelEventName,
							s.GetEventName()), zap.Any(logging.LabelEventSourceType, s.GetEventSourceType()), zap.String("eventID", event.ID()))
						e.metrics.EventSent(s.GetEventSourceName(), s.GetEventName())
						return nil
					})
				}); err != nil {
					logger.Errorw("Failed to start listening eventsource", zap.Any(logging.LabelEventSourceType,
						s.GetEventSourceType()), zap.Any(logging.LabelEventName, s.GetEventName()), zap.Error(err))
				}
			}(server)
		}
	}
	logger.Info("Eventing server started.")

	eventServersWGDone := make(chan bool)
	go func() {
		wg.Wait()
		close(eventServersWGDone)
	}()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down...")
			cancel()
			<-eventServersWGDone
			connWG.Wait()
			return nil
		case <-eventServersWGDone:
			logger.Error("Erroring out, no active event server running")
			cancel()
			connWG.Wait()
			return fmt.Errorf("no active event server running")
		}
	}
}

func generateClientID(hostname string) string {
	randomNum, _ := rand.Int(rand.Reader, big.NewInt(int64(1000)))
	clientID := fmt.Sprintf("client-%s-%v", strings.ReplaceAll(hostname, ".", "_"), randomNum.Int64())
	return clientID
}

func filterEvent(data []byte, filter *aev1.EventSourceFilter) (bool, error) {
	dataMap := make(map[string]interface{})
	err := json.Unmarshal(data, &dataMap)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal data, %w", err)
	}

	params := make(map[string]interface{})
	for key, value := range dataMap {
		params[strings.ReplaceAll(key, "-", "_")] = value
	}
	env := expr.GetFuncMap(params)
	return expr.EvalBool(filter.Expression, env)
}
