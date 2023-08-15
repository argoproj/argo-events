package eventsource

import (
	"context"
	"fmt"

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
	if n == nil {
		return fmt.Errorf("Publish() failed; JetstreamSourceConn is nil")
	}
	return n.STANConn.Publish(n.subject, msg.Body)
}
