package eventsources

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/common/leaderelection"
	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventbus"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	eventsourcecommon "github.com/argoproj/argo-events/eventsources/common"
	"github.com/argoproj/argo-events/eventsources/sources/amqp"
	"github.com/argoproj/argo-events/eventsources/sources/awssns"
	"github.com/argoproj/argo-events/eventsources/sources/awssqs"
	"github.com/argoproj/argo-events/eventsources/sources/azureeventshub"
	"github.com/argoproj/argo-events/eventsources/sources/bitbucket"
	"github.com/argoproj/argo-events/eventsources/sources/bitbucketserver"
	"github.com/argoproj/argo-events/eventsources/sources/calendar"
	"github.com/argoproj/argo-events/eventsources/sources/emitter"
	"github.com/argoproj/argo-events/eventsources/sources/file"
	"github.com/argoproj/argo-events/eventsources/sources/gcppubsub"
	"github.com/argoproj/argo-events/eventsources/sources/generic"
	"github.com/argoproj/argo-events/eventsources/sources/github"
	"github.com/argoproj/argo-events/eventsources/sources/gitlab"
	"github.com/argoproj/argo-events/eventsources/sources/hdfs"
	"github.com/argoproj/argo-events/eventsources/sources/kafka"
	"github.com/argoproj/argo-events/eventsources/sources/minio"
	"github.com/argoproj/argo-events/eventsources/sources/mqtt"
	"github.com/argoproj/argo-events/eventsources/sources/nats"
	"github.com/argoproj/argo-events/eventsources/sources/nsq"
	"github.com/argoproj/argo-events/eventsources/sources/pulsar"
	"github.com/argoproj/argo-events/eventsources/sources/redis"
	"github.com/argoproj/argo-events/eventsources/sources/resource"
	"github.com/argoproj/argo-events/eventsources/sources/slack"
	"github.com/argoproj/argo-events/eventsources/sources/storagegrid"
	"github.com/argoproj/argo-events/eventsources/sources/stripe"
	"github.com/argoproj/argo-events/eventsources/sources/webhook"
	eventsourcemetrics "github.com/argoproj/argo-events/metrics"
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
	StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Options) error) error
}

// GetEventingServers returns the mapping of event source type and list of eventing servers
func GetEventingServers(eventSource *v1alpha1.EventSource, metrics *eventsourcemetrics.Metrics) map[apicommon.EventSourceType][]EventingServer {
	result := make(map[apicommon.EventSourceType][]EventingServer)
	if len(eventSource.Spec.AMQP) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.AMQP {
			servers = append(servers, &amqp.EventListener{EventSourceName: eventSource.Name, EventName: k, AMQPEventSource: v, Metrics: metrics})
		}
		result[apicommon.AMQPEvent] = servers
	}
	if len(eventSource.Spec.AzureEventsHub) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.AzureEventsHub {
			servers = append(servers, &azureeventshub.EventListener{EventSourceName: eventSource.Name, EventName: k, AzureEventsHubEventSource: v, Metrics: metrics})
		}
		result[apicommon.AzureEventsHub] = servers
	}
	if len(eventSource.Spec.Bitbucket) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Bitbucket {
			servers = append(servers, &bitbucket.EventListener{EventSourceName: eventSource.Name, EventName: k, BitbucketEventSource: v, Metrics: metrics})
		}
		result[apicommon.BitbucketEvent] = servers
	}
	if len(eventSource.Spec.BitbucketServer) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.BitbucketServer {
			servers = append(servers, &bitbucketserver.EventListener{EventSourceName: eventSource.Name, EventName: k, BitbucketServerEventSource: v, Metrics: metrics})
		}
		result[apicommon.BitbucketServerEvent] = servers
	}
	if len(eventSource.Spec.Calendar) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Calendar {
			servers = append(servers, &calendar.EventListener{EventSourceName: eventSource.Name, EventName: k, CalendarEventSource: v, Namespace: eventSource.Namespace, Metrics: metrics})
		}
		result[apicommon.CalendarEvent] = servers
	}
	if len(eventSource.Spec.Emitter) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Emitter {
			servers = append(servers, &emitter.EventListener{EventSourceName: eventSource.Name, EventName: k, EmitterEventSource: v, Metrics: metrics})
		}
		result[apicommon.EmitterEvent] = servers
	}
	if len(eventSource.Spec.File) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.File {
			servers = append(servers, &file.EventListener{EventSourceName: eventSource.Name, EventName: k, FileEventSource: v, Metrics: metrics})
		}
		result[apicommon.FileEvent] = servers
	}
	if len(eventSource.Spec.Github) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Github {
			servers = append(servers, &github.EventListener{EventSourceName: eventSource.Name, EventName: k, GithubEventSource: v, Metrics: metrics})
		}
		result[apicommon.GithubEvent] = servers
	}
	if len(eventSource.Spec.Gitlab) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Gitlab {
			servers = append(servers, &gitlab.EventListener{EventSourceName: eventSource.Name, EventName: k, GitlabEventSource: v, Metrics: metrics})
		}
		result[apicommon.GitlabEvent] = servers
	}
	if len(eventSource.Spec.HDFS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.HDFS {
			servers = append(servers, &hdfs.EventListener{EventSourceName: eventSource.Name, EventName: k, HDFSEventSource: v, Metrics: metrics})
		}
		result[apicommon.HDFSEvent] = servers
	}
	if len(eventSource.Spec.Kafka) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Kafka {
			servers = append(servers, &kafka.EventListener{EventSourceName: eventSource.Name, EventName: k, KafkaEventSource: v, Metrics: metrics})
		}
		result[apicommon.KafkaEvent] = servers
	}
	if len(eventSource.Spec.MQTT) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.MQTT {
			servers = append(servers, &mqtt.EventListener{EventSourceName: eventSource.Name, EventName: k, MQTTEventSource: v, Metrics: metrics})
		}
		result[apicommon.MQTTEvent] = servers
	}
	if len(eventSource.Spec.Minio) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Minio {
			servers = append(servers, &minio.EventListener{EventSourceName: eventSource.Name, EventName: k, MinioEventSource: v, Metrics: metrics})
		}
		result[apicommon.MinioEvent] = servers
	}
	if len(eventSource.Spec.NATS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.NATS {
			servers = append(servers, &nats.EventListener{EventSourceName: eventSource.Name, EventName: k, NATSEventSource: v, Metrics: metrics})
		}
		result[apicommon.NATSEvent] = servers
	}
	if len(eventSource.Spec.NSQ) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.NSQ {
			servers = append(servers, &nsq.EventListener{EventSourceName: eventSource.Name, EventName: k, NSQEventSource: v, Metrics: metrics})
		}
		result[apicommon.NSQEvent] = servers
	}
	if len(eventSource.Spec.PubSub) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.PubSub {
			servers = append(servers, &gcppubsub.EventListener{EventSourceName: eventSource.Name, EventName: k, PubSubEventSource: v, Metrics: metrics})
		}
		result[apicommon.PubSubEvent] = servers
	}
	if len(eventSource.Spec.Redis) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Redis {
			servers = append(servers, &redis.EventListener{EventSourceName: eventSource.Name, EventName: k, RedisEventSource: v, Metrics: metrics})
		}
		result[apicommon.RedisEvent] = servers
	}
	if len(eventSource.Spec.SNS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.SNS {
			servers = append(servers, &awssns.EventListener{EventSourceName: eventSource.Name, EventName: k, SNSEventSource: v, Metrics: metrics})
		}
		result[apicommon.SNSEvent] = servers
	}
	if len(eventSource.Spec.SQS) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.SQS {
			servers = append(servers, &awssqs.EventListener{EventSourceName: eventSource.Name, EventName: k, SQSEventSource: v, Metrics: metrics})
		}
		result[apicommon.SQSEvent] = servers
	}
	if len(eventSource.Spec.Slack) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Slack {
			servers = append(servers, &slack.EventListener{EventSourceName: eventSource.Name, EventName: k, SlackEventSource: v, Metrics: metrics})
		}
		result[apicommon.SlackEvent] = servers
	}
	if len(eventSource.Spec.StorageGrid) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.StorageGrid {
			servers = append(servers, &storagegrid.EventListener{EventSourceName: eventSource.Name, EventName: k, StorageGridEventSource: v, Metrics: metrics})
		}
		result[apicommon.StorageGridEvent] = servers
	}
	if len(eventSource.Spec.Stripe) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Stripe {
			servers = append(servers, &stripe.EventListener{EventSourceName: eventSource.Name, EventName: k, StripeEventSource: v, Metrics: metrics})
		}
		result[apicommon.StripeEvent] = servers
	}
	if len(eventSource.Spec.Webhook) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Webhook {
			servers = append(servers, &webhook.EventListener{EventSourceName: eventSource.Name, EventName: k, WebhookContext: v, Metrics: metrics})
		}
		result[apicommon.WebhookEvent] = servers
	}
	if len(eventSource.Spec.Resource) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Resource {
			servers = append(servers, &resource.EventListener{EventSourceName: eventSource.Name, EventName: k, ResourceEventSource: v, Metrics: metrics})
		}
		result[apicommon.ResourceEvent] = servers
	}
	if len(eventSource.Spec.Pulsar) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Pulsar {
			servers = append(servers, &pulsar.EventListener{EventSourceName: eventSource.Name, EventName: k, PulsarEventSource: v, Metrics: metrics})
		}
		result[apicommon.PulsarEvent] = servers
	}
	if len(eventSource.Spec.Generic) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Generic {
			servers = append(servers, &generic.EventListener{EventSourceName: eventSource.Name, EventName: k, GenericEventSource: v, Metrics: metrics})
		}
		result[apicommon.GenericEvent] = servers
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

	metrics *eventsourcemetrics.Metrics
}

// NewEventSourceAdaptor returns a new EventSourceAdaptor
func NewEventSourceAdaptor(eventSource *v1alpha1.EventSource, eventBusConfig *eventbusv1alpha1.BusConfig, eventBusSubject, hostname string, metrics *eventsourcemetrics.Metrics) *EventSourceAdaptor {
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

	recreateTypes := make(map[apicommon.EventSourceType]bool)
	for _, esType := range apicommon.RecreateStrategyEventSources {
		recreateTypes[esType] = true
	}
	isRecreatType := false
	servers := GetEventingServers(e.eventSource, e.metrics)
	for k := range servers {
		if _, ok := recreateTypes[k]; ok {
			isRecreatType = true
		}
		// This is based on the presumption that all the events in one
		// EventSource object use the same type of deployment strategy
		break
	}
	if !isRecreatType {
		return e.run(ctx, servers)
	}

	custerName := fmt.Sprintf("%s-eventsource-%s", e.eventSource.Namespace, e.eventSource.Name)
	elector, err := leaderelection.NewEventBusElector(ctx, *e.eventBusConfig, custerName, int(e.eventSource.Spec.GetReplicas()))
	if err != nil {
		log.Errorw("failed to get an elector", zap.Error(err))
		return err
	}
	elector.RunOrDie(ctx, leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			if err := e.run(ctx, servers); err != nil {
				log.Fatalw("failed to start", zap.Error(err))
			}
		},
		OnStoppedLeading: func() {
			log.Fatalf("leader lost: %s", e.hostname)
		},
	})

	return nil
}

func (e *EventSourceAdaptor) run(ctx context.Context, servers map[apicommon.EventSourceType][]EventingServer) error {
	logger := logging.FromContext(ctx)
	logger.Info("Starting event source server...")
	clientID := generateClientID(e.hostname)
	driver, err := eventbus.GetDriver(ctx, *e.eventBusConfig, e.eventBusSubject, clientID)
	if err != nil {
		logger.Errorw("failed to get eventbus driver", zap.Error(err))
		return err
	}
	if err = common.Connect(&common.DefaultBackoff, func() error {
		e.eventBusConn, err = driver.Connect()
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
					driver, err := eventbus.GetDriver(ctx, *e.eventBusConfig, e.eventBusSubject, clientID)
					if err != nil {
						logger.Errorw("failed to get eventbus driver during reconnection", zap.Error(err))
						continue
					}
					e.eventBusConn, err = driver.Connect()
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
			wg.Add(1)
			go func(s EventingServer) {
				defer wg.Done()
				e.metrics.IncRunningServices(s.GetEventSourceName())
				defer e.metrics.DecRunningServices(s.GetEventSourceName())
				duration := apicommon.FromString("1s")
				factor := apicommon.NewAmount("1")
				jitter := apicommon.NewAmount("30")
				backoff := apicommon.Backoff{
					Steps:    10,
					Duration: &duration,
					Factor:   &factor,
					Jitter:   &jitter,
				}
				if err = common.Connect(&backoff, func() error {
					return s.StartListening(ctx, func(data []byte, opts ...eventsourcecommon.Options) error {
						event := cloudevents.NewEvent()
						event.SetID(fmt.Sprintf("%x", uuid.New()))
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
							return errors.New("failed to publish event, eventbus connection closed")
						}
						if err = driver.Publish(e.eventBusConn, eventBody); err != nil {
							logger.Errorw("failed to publish an event", zap.Error(err), zap.String(logging.LabelEventName,
								s.GetEventName()), zap.Any(logging.LabelEventSourceType, s.GetEventSourceType()))
							e.metrics.EventSentFailed(s.GetEventSourceName(), s.GetEventName())
							return err
						}
						logger.Infow("succeeded to publish an event", zap.String(logging.LabelEventName,
							s.GetEventName()), zap.Any(logging.LabelEventSourceType, s.GetEventSourceType()), zap.String("eventID", event.ID()))
						e.metrics.EventSent(s.GetEventSourceName(), s.GetEventName())
						return nil
					})
				}); err != nil {
					logger.Errorw("failed to start listening eventsource", zap.Any(logging.LabelEventSourceType,
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
			return errors.New("no active event server running")
		}
	}
}

func generateClientID(hostname string) string {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	clientID := fmt.Sprintf("client-%s-%v", strings.ReplaceAll(hostname, ".", "_"), r1.Intn(1000))
	return clientID
}
