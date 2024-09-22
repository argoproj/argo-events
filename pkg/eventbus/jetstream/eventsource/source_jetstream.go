package eventsource

import (
	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
	jetstreambase "github.com/argoproj/argo-events/pkg/eventbus/jetstream/base"
	"go.uber.org/zap"
)

type SourceJetstream struct {
	*jetstreambase.Jetstream
	eventSourceName string
}

func NewSourceJetstream(url, eventSourceName string, streamConfig string, auth *eventbuscommon.Auth, logger *zap.SugaredLogger) (*SourceJetstream, error) {
	baseJetstream, err := jetstreambase.NewJetstream(url, streamConfig, auth, logger)
	if err != nil {
		return nil, err
	}
	return &SourceJetstream{
		baseJetstream,
		eventSourceName,
	}, nil
}

func (n *SourceJetstream) Initialize() error {
	return n.Init() // member of jetstreambase.Jetstream
}

func (n *SourceJetstream) Connect(clientID string) (eventbuscommon.EventSourceConnection, error) {
	conn, err := n.MakeConnection()
	if err != nil {
		return nil, err
	}

	return CreateJetstreamSourceConn(conn, n.eventSourceName), nil
}
