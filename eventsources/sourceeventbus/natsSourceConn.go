package sourceeventbus

import (
	"context"

	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
)

type NATSStreamingSourceConn struct {
	*eventbusdriver.NATSStreamingConnection
	eventSourceName string
	subject         string
}

func (n *NATSStreamingSourceConn) PublishEvent(ctx context.Context,
	evt eventbusdriver.Event,
	message []byte) error {
	return n.Publish(n.subject, message)
}
