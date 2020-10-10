package generic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// EventListener implements Eventing for generic event source
type EventListener struct {
	EventSourceName    string
	EventName          string
	GenericEventSource v1alpha1.GenericEventSource
	conn               *grpc.ClientConn
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
func (el *EventListener) GetEventSourceType() apicommon.EventSourceType {
	return apicommon.GenericEvent
}

// StartListening listens to generic events
func (el *EventListener) StartListening(ctx context.Context, dispatch func([]byte) error) error {
	logger := logging.FromContext(ctx).
		With(zap.String(logging.LabelEventSourceType, string(el.GetEventSourceType())),
			zap.String(logging.LabelEventName, el.GetEventName()),
			zap.String("url", el.GenericEventSource.URL))
	logger.Info("started processing the generic event source...")
	defer sources.Recover(el.GetEventName())

	logger.Info("connecting to eventsource server in 5 seconds...")
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			logger.Info("closing client connection and exiting eventsource...")
			el.conn.Close()
			return nil
		case <-ticker.C:
			if el.conn == nil || el.conn.GetState() == connectivity.Shutdown || el.conn.GetState() == connectivity.TransientFailure {
				logger.Info("dialing eventsource server...")
				eventStream, err := el.connect()
				if err != nil {
					logger.Error("failed to reconnect eventsource server, reconnecting in 5 seconds...", zap.Error(err))
					continue
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
					logger.Info("received an event from server")
					eventData := &events.GenericEventData{
						Metadata: el.GenericEventSource.Metadata,
					}
					if el.GenericEventSource.JSONBody {
						eventData.Body = (*json.RawMessage)(&event.Payload)
					}
					eventBytes, err := json.Marshal(eventData)
					if err != nil {
						logger.Errorw("failed to marshal the event data", zap.Error(err))
						continue
					}
					logger.Info("dispatching event...")
					if err := dispatch(eventBytes); err != nil {
						logger.Errorw("failed to dispatch the event", zap.Error(err))
					}
				}
			}
		}
	}
}

func (el *EventListener) connect() (Eventing_StartEventSourceClient, error) {
	var opt []grpc.DialOption
	opt = append(opt, grpc.WithBlock())
	if el.GenericEventSource.Insecure {
		opt = append(opt, grpc.WithInsecure())
	}
	conn, err := grpc.DialContext(context.Background(), el.GenericEventSource.URL, opt...)
	if err != nil {
		return nil, err
	}
	el.conn = conn
	client := NewEventingClient(el.conn)
	return client.StartEventSource(context.Background(), &EventSource{
		Name:   el.GetEventName(),
		Config: []byte(el.GenericEventSource.Config),
	})
}
