package eventsource

import (
	"context"
	"fmt"

	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	jetstreambase "github.com/argoproj/argo-events/eventbus/jetstream/base"
)

type JetstreamSourceConn struct {
	*jetstreambase.JetstreamConnection
	eventSourceName string
}

func (n *JetstreamSourceConn) PublishEvent(ctx context.Context,
	evt eventbuscommon.Event,
	message []byte) error {
	// derive subject from event source name and event name
	subject := fmt.Sprintf("default.%s.%s", evt.EventSourceName, evt.EventName)
	return n.Publish(subject, message)
}
