package driver

import (
	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type JetstreamConnection struct {
	natsConn  *nats.Conn
	jsContext nats.JetStreamContext

	natsConnected bool
	clientID      string // seems like jetstream doesn't have this notion; we can just have this to uniquely identify ourselves in the log

	logger *zap.SugaredLogger
}

func (jsc *JetstreamConnection) Close() error {

	if jsc.natsConn != nil && jsc.natsConn.IsConnected() {
		jsc.natsConn.Close()
	}
	return nil
}

func (jsc *JetstreamConnection) IsClosed() bool {
	if jsc.natsConn == nil || !jsc.natsConnected || jsc.natsConn.IsClosed() {
		return true
	}
	return false
}

func (jsc *JetstreamConnection) Publish(subject string, data []byte) error {
	// todo: On the publishing side you can avoid duplicate message ingestion using the Message Deduplication feature.
	_, err := jsc.jsContext.Publish(subject, data)
	return err
}

func (jsc *JetstreamConnection) ClientID() string {
	return jsc.clientID
}
