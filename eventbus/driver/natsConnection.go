package driver

import (
	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"go.uber.org/zap"
)

type NATSStreamingConnection struct {
	natsConn *nats.Conn
	stanConn stan.Conn

	natsConnected bool
	stanConnected bool

	//subject  string
	clientID string

	logger *zap.SugaredLogger
}

func (nsc *NATSStreamingConnection) Close() error {
	if nsc.stanConn != nil {
		err := nsc.stanConn.Close()
		if err != nil {
			return err
		}
	}
	if nsc.natsConn != nil && nsc.natsConn.IsConnected() {
		nsc.natsConn.Close()
	}
	return nil
}

func (nsc *NATSStreamingConnection) IsClosed() bool {
	if nsc.natsConn == nil || nsc.stanConn == nil || !nsc.natsConnected || !nsc.stanConnected || nsc.natsConn.IsClosed() {
		return true
	}
	return false
}

func (nsc *NATSStreamingConnection) Publish(subject string, data []byte) error {
	return nsc.stanConn.Publish(subject, data)
}

func (nsc *NATSStreamingConnection) ClientID() string {
	return nsc.clientID
}
