package base

import (
	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"go.uber.org/zap"
)

type STANConnection struct {
	NATSConn *nats.Conn
	STANConn stan.Conn

	NATSConnected bool
	STANConnected bool

	// defaultSubject  string
	clientID string

	Logger *zap.SugaredLogger
}

func (nsc *STANConnection) Close() error {
	if nsc.STANConn != nil {
		err := nsc.STANConn.Close()
		if err != nil {
			return err
		}
	}
	if nsc.NATSConn != nil && nsc.NATSConn.IsConnected() {
		nsc.NATSConn.Close()
	}
	return nil
}

func (nsc *STANConnection) IsClosed() bool {
	if nsc.NATSConn == nil || nsc.STANConn == nil || !nsc.NATSConnected || !nsc.STANConnected || nsc.NATSConn.IsClosed() {
		return true
	}
	return false
}

func (nsc *STANConnection) Publish(subject string, data []byte) error {
	return nsc.STANConn.Publish(subject, data)
}

func (nsc *STANConnection) ClientID() string {
	return nsc.clientID
}
