package eventsource

import (
	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
	stanbase "github.com/argoproj/argo-events/pkg/eventbus/stan/base"
	"go.uber.org/zap"
)

type SourceSTAN struct {
	*stanbase.STAN
	eventSourceName string
	subject         string
}

func NewSourceSTAN(url, clusterID, eventSourceName string, subject string, auth *eventbuscommon.Auth, logger *zap.SugaredLogger) *SourceSTAN {
	return &SourceSTAN{
		stanbase.NewSTAN(url, clusterID, auth, logger),
		eventSourceName,
		subject,
	}
}

func (n *SourceSTAN) Initialize() error {
	return nil
}

func (n *SourceSTAN) Connect(clientID string) (eventbuscommon.EventSourceConnection, error) {
	conn, err := n.MakeConnection(clientID)
	if err != nil {
		return nil, err
	}

	return &STANSourceConn{conn, n.eventSourceName, n.subject}, nil
}
