package generic

import (
	"context"
	"encoding/json"
	fmt "fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventsourcecommon "github.com/argoproj/argo-events/pkg/eventsources/common"
	"github.com/argoproj/argo-events/pkg/eventsources/events"
	"github.com/argoproj/argo-events/pkg/eventsources/sources"
	metrics "github.com/argoproj/argo-events/pkg/metrics"
	"github.com/argoproj/argo-events/pkg/shared/logging"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// EventListener implements Eventing for generic event source
type EventListener struct {
	EventSourceName    string
	EventName          string
	GenericEventSource v1alpha1.GenericEventSource
	Metrics            *metrics.Metrics

	conn *grpc.ClientConn
}

// GetEventSourceName returns name of event source
func (el *EventListener) GetEventSourceName() string {
	return el.EventSourceName
}

// GetEventName returns name of event
func (el *EventListener) GetEventName() string {
	return el.EventName
}

// GetEventSourceType return type of event server
func (el *EventListener) GetEventSourceType() v1alpha1.EventSourceType {
	return v1alpha1.GenericEvent
}

// StartListening listens to generic events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte, ...eventsourcecommon.Option) error) error {
	logger := logging.FromContext(ctx).
		With(zap.String(logging.LabelEventSourceType, string(el.GetEventSourceType())),
			zap.String(logging.LabelEventName, el.GetEventName()),
			zap.String("url", el.GenericEventSource.URL))
	logger.Info("started processing the generic event source...")
	defer sources.Recover(el.GetEventName())

	logger.Info("connecting to eventsource server in 5 seconds...")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("closing client connection and exiting eventsource...")
			if el.conn != nil && (el.conn.GetState() == connectivity.Ready || el.conn.GetState() == connectivity.Connecting) {
				el.conn.Close()
			}
			return nil
		case <-ticker.C:
			var eventStream Eventing_StartEventSourceClient
			if el.conn == nil || el.conn.GetState() == connectivity.Shutdown || el.conn.GetState() == connectivity.TransientFailure {
				logger.Info("dialing eventsource server...")
				var err error
				eventStream, err = el.connect(logger)
				if err != nil {
					logger.Errorw("failed to connect eventsource server, reconnecting in 5 seconds...", zap.Error(err))
					continue
				}
			}
			if el.conn.GetState() == connectivity.Ready || el.conn.GetState() == connectivity.Idle {
				if eventStream == nil {
					var err error
					eventStream, err = el.buildEventStream()
					if err != nil {
						logger.Errorw("failed to build stream, retrying in 5 seconds...", zap.Error(err),
							zap.String("ess-connection-state", el.conn.GetState().String()))
						continue
					}
				}
				logger.Info("connected to eventsource server successfully, started event stream...")
				for {
					event, err := eventStream.Recv()
					if err != nil {
						logger.Errorw("failed to receive events from the event stream, reconnecting in 5 seconds...", zap.Error(err))
						// close the connection and retry in next cycle.
						el.conn.Close()
						break
					}
					if err := el.handleOne(event, dispatch, logger); err != nil {
						logger.Errorw("failed to process a Generics event", zap.Error(err))
						el.Metrics.EventProcessingFailed(el.GetEventSourceName(), el.GetEventName())
					}
				}
			} else {
				logger.Warnw("eventsource server connection in an incorrect state, retrying in 5 seconds...",
					zap.String("ess-connection-state", el.conn.GetState().String()))
			}
		}
	}
}

func (el *EventListener) handleOne(event *Event, dispatch func([]byte, ...eventsourcecommon.Option) error, logger *zap.SugaredLogger) error {
	defer func(start time.Time) {
		el.Metrics.EventProcessingDuration(el.GetEventSourceName(), el.GetEventName(), float64(time.Since(start)/time.Millisecond))
	}(time.Now())

	logger.Info("received an event from server")
	eventData := &events.GenericEventData{
		Metadata: el.GenericEventSource.Metadata,
	}
	if el.GenericEventSource.JSONBody {
		eventData.Body = (*json.RawMessage)(&event.Payload)
	} else {
		eventData.Body = event.Payload
	}
	eventBytes, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal the event data, %w", err)
	}
	logger.Info("dispatching event...")
	if err := dispatch(eventBytes); err != nil {
		return fmt.Errorf("failed to dispatch a Generic event, %w", err)
	}
	return nil
}

func (el *EventListener) connect(logger *zap.SugaredLogger) (Eventing_StartEventSourceClient, error) {
	var opt []grpc.DialOption
	if el.GenericEventSource.Insecure {
		opt = append(opt, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	conn, err := grpc.NewClient(el.GenericEventSource.URL, opt...)
	if err != nil {
		logger.Errorw("failed to inititalise gRPC stream to eventsource server", zap.Error(err))
		return nil, err
	}
	el.conn = conn
	return el.buildEventStream()
}

func (el *EventListener) buildEventStream() (Eventing_StartEventSourceClient, error) {
	client := NewEventingClient(el.conn)
	ctx := context.Background()
	if el.GenericEventSource.AuthSecret != nil {
		token, err := sharedutil.GetSecretFromVolume(el.GenericEventSource.AuthSecret)
		if err != nil {
			return nil, err
		}
		auth := fmt.Sprintf("Bearer %s", token)
		ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", auth))
	}
	return client.StartEventSource(ctx, &EventSource{
		Name:   el.GetEventName(),
		Config: []byte(el.GenericEventSource.Config),
	})
}
