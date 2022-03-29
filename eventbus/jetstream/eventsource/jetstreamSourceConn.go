package eventsource

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	jetstreambase "github.com/argoproj/argo-events/eventbus/jetstream/base"
	nats "github.com/nats-io/nats.go"
)

type JetstreamSourceConn struct {
	*jetstreambase.JetstreamConnection
	eventSourceName string
	randomGenerator *rand.Rand
}

func CreateJetstreamSourceConn(conn *jetstreambase.JetstreamConnection, eventSourceName string) *JetstreamSourceConn {
	randSource := rand.NewSource(time.Now().UnixNano())
	randomGenerator := rand.New(randSource)

	return &JetstreamSourceConn{
		conn, eventSourceName, randomGenerator,
	}
}

func (jsc *JetstreamSourceConn) Publish(ctx context.Context,
	evt eventbuscommon.Event,
	message []byte) error {
	// exactly once on the publishing side is done by assigning a "deduplication key" to the message
	dedupKey := nats.MsgId(strconv.Itoa(jsc.randomGenerator.Intn(1000000)))

	// derive subject from event source name and event name
	subject := fmt.Sprintf("default.%s.%s", evt.EventSourceName, evt.EventName)
	_, err := jsc.JSContext.Publish(subject, message, dedupKey)
	jsc.Logger.Debugf("published message to subject %s", subject)
	return err
}
