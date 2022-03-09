package sourceeventbus

import (
	"context"
	"fmt"

	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
)

type JetstreamSourceConn struct {
	*eventbusdriver.JetstreamConnection
	eventSourceName string
}

func (n *JetstreamSourceConn) PublishEvent(ctx context.Context,
	evt eventbusdriver.Event,
	message []byte,
	defaultSubject *string) error {

	// derive subject from event source name and event name
	subject := fmt.Sprintf("default-%s-%s", evt.EventSourceName, evt.EventName)
	n.Publish(subject, message)

	return nil
}
