package eventsources

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventbus"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	"github.com/argoproj/argo-events/eventsources/sources/amqp"
	"github.com/argoproj/argo-events/eventsources/sources/awssns"
	"github.com/argoproj/argo-events/eventsources/sources/awssqs"
	"github.com/argoproj/argo-events/eventsources/sources/calendar"
	"github.com/argoproj/argo-events/eventsources/sources/webhook"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// EventingServer is the server API for Eventing service.
type EventingServer interface {

	// ValidateEventSource validates an event source.
	ValidateEventSource(context.Context) error

	GetEventSourceName() string

	GetEventName() string

	GetEventSourceType() apicommon.EventSourceType

	// Function to start listening events.
	StartListening(ctx context.Context, stopCh <-chan struct{}, dispatch func([]byte) error) error
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

	}
	if len(eventSource.Spec.Calendar) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Calendar {
			servers = append(servers, &calendar.EventListener{EventSourceName: eventSource.Name, EventName: k, CalendarEventSource: v})
		}
		result[apicommon.CalendarEvent] = servers
	}
	if len(eventSource.Spec.Emitter) != 0 {

	}
	if len(eventSource.Spec.File) != 0 {

	}
	if len(eventSource.Spec.Generic) != 0 {

	}
	if len(eventSource.Spec.Github) != 0 {

	}
	if len(eventSource.Spec.Gitlab) != 0 {

	}
	if len(eventSource.Spec.HDFS) != 0 {

	}
	if len(eventSource.Spec.Kafka) != 0 {

	}
	if len(eventSource.Spec.MQTT) != 0 {

	}
	if len(eventSource.Spec.Minio) != 0 {

	}
	if len(eventSource.Spec.NATS) != 0 {

	}
	if len(eventSource.Spec.NSQ) != 0 {

	}
	if len(eventSource.Spec.PubSub) != 0 {

	}
	if len(eventSource.Spec.Redis) != 0 {

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

	}
	if len(eventSource.Spec.StorageGrid) != 0 {

	}
	if len(eventSource.Spec.Stripe) != 0 {

	}
	if len(eventSource.Spec.Webhook) != 0 {
		servers := []EventingServer{}
		for k, v := range eventSource.Spec.Webhook {
			servers = append(servers, &webhook.EventListener{EventSourceName: eventSource.Name, EventName: k, WebhookContext: v})
		}
		result[apicommon.WebhookEvent] = servers
	}
	return result
}

// EventSourceAdaptor is the adaptor for eventsource service
type EventSourceAdaptor struct {
	// kubeClient is the kubernetes client
	kubeClient kubernetes.Interface
	// clientPool manages a pool of dynamic clients.
	dynamicClient dynamic.Interface

	eventSource     *v1alpha1.EventSource
	eventBusConfig  *eventbusv1alpha1.BusConfig
	eventBusSubject string
	hostname        string

	eventBusConn eventbusdriver.Connection
}

// NewEventSourceAdaptor returns a new EventSourceAdaptor
func NewEventSourceAdaptor(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, eventSource *v1alpha1.EventSource, eventBusConfig *eventbusv1alpha1.BusConfig, eventBusSubject, hostname string) *EventSourceAdaptor {
	return &EventSourceAdaptor{
		kubeClient:      kubeClient,
		dynamicClient:   dynamicClient,
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
	for _, ss := range servers {
		for _, server := range ss {
			err := server.ValidateEventSource(cctx)
			if err != nil {
				logger.WithError(err).Errorln("Validation failed.")
				return err
			}
			go func(s EventingServer) {
				err := s.StartListening(cctx, stopCh, func(data []byte) error {
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
					return driver.Publish(e.eventBusConn, eventBody)
				})
				logger.WithError(err).Errorln("failed to start service.")
			}(server)
		}
	}
	logger.Info("Eventing server started.")
	<-stopCh
	logger.Info("Shutting down...")
	return nil
}
