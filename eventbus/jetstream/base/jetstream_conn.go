package base

import (
	"errors"

	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type JetstreamConnection struct {
	NATSConn  *nats.Conn
	JSContext nats.JetStreamContext

	NATSConnected bool

	Logger *zap.SugaredLogger
}

func (jsc *JetstreamConnection) Close() error {
	if jsc == nil {
		return errors.New("can't close Jetstream connection, JetstreamConnection is nil")
	}
	if jsc.NATSConn != nil && jsc.NATSConn.IsConnected() {
		jsc.NATSConn.Close()
	}
	return nil
}

func (jsc *JetstreamConnection) IsClosed() bool {
	return jsc == nil || jsc.NATSConn == nil || !jsc.NATSConnected || jsc.NATSConn.IsClosed()
}
