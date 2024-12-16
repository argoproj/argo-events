package eventsource

import (
	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
	jetstreambase "github.com/argoproj/argo-events/pkg/eventbus/jetstream/base"
	"go.uber.org/zap"
)

type SourceJetstream struct {
	*jetstreambase.Jetstream
	eventSourceName string
}

func NewSourceJetstream(url, eventSourceName string, streamConfig string, auth *eventbuscommon.Auth, logger *zap.SugaredLogger, tls *v1alpha1.TLSConfig) (*SourceJetstream, error) {
	baseJetstream, err := jetstreambase.NewJetstream(url, streamConfig, auth, logger, tls)
	if err != nil {
		return nil, err
	}
	return &SourceJetstream{
		baseJetstream,
		eventSourceName,
	}, nil
}

func (n *SourceJetstream) Initialize() error {
	return n.Jetstream.Init() // member of jetstreambase.Jetstream
}

func (n *SourceJetstream) Connect(clientID string) (eventbuscommon.EventSourceConnection, error) {
	conn, err := n.Jetstream.MakeConnection()
	if err != nil {
		return nil, err
	}

	return CreateJetstreamSourceConn(conn, n.eventSourceName), nil
}
