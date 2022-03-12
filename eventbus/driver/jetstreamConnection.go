package driver

import (
	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type JetstreamConnection struct {
	NATSConn  *nats.Conn
	JSContext nats.JetStreamContext

	NATSConnected bool
	clientID      string // seems like jetstream doesn't have this notion; we can just have this to uniquely identify ourselves in the log (todo: consider this further)

	Logger *zap.SugaredLogger
}

func (jsc *JetstreamConnection) Close() error {
	if jsc.NATSConn != nil && jsc.NATSConn.IsConnected() {
		jsc.NATSConn.Close()
	}
	return nil
}

func (jsc *JetstreamConnection) IsClosed() bool {
	if jsc.NATSConn == nil || !jsc.NATSConnected || jsc.NATSConn.IsClosed() {
		return true
	}
	return false
}

func (jsc *JetstreamConnection) Publish(subject string, data []byte) error {
	// todo: On the publishing side you can avoid duplicate message ingestion using the Message Deduplication feature.
	_, err := jsc.JSContext.Publish(subject, data)
	return err
}

func (jsc *JetstreamConnection) ClientID() string {
	return jsc.clientID
}
