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

func (jsc *JetstreamSourceConn) Publish(ctx context.Context,
	evt eventbuscommon.Event,
	message []byte) error {
	// todo: On the publishing side you can avoid duplicate message ingestion using the Message Deduplication feature.

	// derive subject from event source name and event name
	subject := fmt.Sprintf("default.%s.%s", evt.EventSourceName, evt.EventName)
	_, err := jsc.JSContext.Publish(subject, message)
	return err
}
