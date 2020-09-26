package generic

import (
	"context"
	"encoding/json"
	"io"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/eventsources/sources"
	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/events"
	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// EventListener implements Eventing for generic event source
type EventListener struct {
	EventSourceName    string
	EventName          string
	GenericEventSource v1alpha1.GenericEventSource
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

	var opt []grpc.DialOption
	opt = append(opt, grpc.WithBlock())
	if el.GenericEventSource.Insecure {
		opt = append(opt, grpc.WithInsecure())
	}

	logger.Info("dialing gRPC server...")
	clientConn, err := grpc.DialContext(ctx, el.GenericEventSource.URL, opt...)
	if err != nil {
		return errors.Wrapf(err, "failed to prepare gRPC client for %s", el.GetEventName())
	}

	client := NewEventingClient(clientConn)

	logger.Info("starting event stream...")
	eventStream, err := client.StartEventSource(ctx, &EventSource{
		Name:   el.GetEventName(),
		Config: []byte(el.GenericEventSource.Config),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to start the event source stream for %s", el.GetEventName())
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("event source is stopped")
			clientConn.Close()
			return nil

		default:
			event, err := eventStream.Recv()
			if err != nil {
				if err == io.EOF {
					logger.Info("event source is stopped by the server")
					return nil
				}

				clientConn.Close()
				return errors.Wrapf(err, "failed to receive event stream from %s", el.GetEventName())
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
				logger.Infow("failed to dispatch the event", zap.Error(err))
				continue
			}
		}
	}
}
