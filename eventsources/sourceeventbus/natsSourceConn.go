package sourceeventbus

import (
	"context"
	"errors"

	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
)

type NATSStreamingSourceConn struct {
	*eventbusdriver.NATSStreamingConnection
	eventSourceName string
}

func (n *NATSStreamingSourceConn) PublishEvent(ctx context.Context,
	evt eventbusdriver.Event,
	message []byte,
	defaultSubject *string) error {
	if defaultSubject == nil {
		return errors.New("can't publish event, default subject not specified")
	}
	n.Publish(*defaultSubject, message)

	return nil
}
