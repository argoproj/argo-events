package eventsource

import (
	"context"
	"fmt"
	"time"

	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
	jetstreambase "github.com/argoproj/argo-events/pkg/eventbus/jetstream/base"
	nats "github.com/nats-io/nats.go"
)

type JetstreamSourceConn struct {
	*jetstreambase.JetstreamConnection
	eventSourceName string
}

func CreateJetstreamSourceConn(conn *jetstreambase.JetstreamConnection, eventSourceName string) *JetstreamSourceConn {
	return &JetstreamSourceConn{
		conn, eventSourceName,
	}
}

func (jsc *JetstreamSourceConn) Publish(ctx context.Context,
	msg eventbuscommon.Message) error {
	if jsc == nil {
		return fmt.Errorf("Publish() failed; JetstreamSourceConn is nil")
	}

	// exactly once on the publishing side is done by assigning a "deduplication key" to the message
	dedupKey := nats.MsgId(msg.ID)

	// derive subject from event source name and event name
	subject := fmt.Sprintf("default.%s.%s", msg.EventSourceName, msg.EventName)

	// The total time including retries should remain under 60 seconds (below is 55)
	_, err := jsc.JSContext.Publish(subject, msg.Body, dedupKey, nats.AckWait(10*time.Second), nats.RetryAttempts(3), nats.RetryWait(time.Second*5))
	jsc.Logger.Debugf("published message to subject %s", subject)
	return err
}

func (conn *JetstreamSourceConn) IsClosed() bool {
	return conn == nil || conn.JetstreamConnection.IsClosed()
}

func (conn *JetstreamSourceConn) Close() error {
	if conn == nil {
		return fmt.Errorf("can't close Jetstream source connection, JetstreamSourceConn is nil")
	}
	return conn.JetstreamConnection.Close()
}
