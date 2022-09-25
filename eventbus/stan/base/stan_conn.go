package base

import (
	"fmt"

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
	ClientID string

	Logger *zap.SugaredLogger
}

func (nsc *STANConnection) Close() error {
	if nsc == nil {
		return fmt.Errorf("can't close STAN connection, STANConnection is nil")
	}
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
	return nsc == nil || nsc.NATSConn == nil || nsc.STANConn == nil || !nsc.NATSConnected || !nsc.STANConnected || nsc.NATSConn.IsClosed()
}
