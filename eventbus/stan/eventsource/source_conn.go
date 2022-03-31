package eventsource

import (
	"context"

	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	stanbase "github.com/argoproj/argo-events/eventbus/stan/base"
)

type STANSourceConn struct {
	*stanbase.STANConnection
	eventSourceName string
	subject         string
}

func (n *STANSourceConn) Publish(ctx context.Context,
	msg eventbuscommon.Message) error {
	return n.STANConn.Publish(n.subject, msg.Body)
}
