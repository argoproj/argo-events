package eventsource

import (
	"context"
	"fmt"

	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
	stanbase "github.com/argoproj/argo-events/pkg/eventbus/stan/base"
)

type STANSourceConn struct {
	*stanbase.STANConnection
	eventSourceName string
	subject         string
}

func (n *STANSourceConn) Publish(ctx context.Context,
	msg eventbuscommon.Message) error {
	if n == nil {
		return fmt.Errorf("Publish() failed; JetstreamSourceConn is nil")
	}
	return n.STANConnection.STANConn.Publish(n.subject, msg.Body)
}

func (conn *STANSourceConn) IsClosed() bool {
	return conn == nil || conn.STANConnection.IsClosed()
}

func (conn *STANSourceConn) Close() error {
	if conn == nil {
		return fmt.Errorf("can't close STAN source connection, STANSourceConn is nil")
	}
	return conn.STANConnection.Close()
}
