package sourceeventbus

import (
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	"go.uber.org/zap"
)

type NATSStreaming struct {
	*eventbusdriver.NATStreaming
	eventSourceName string
	subject         string
}

func NewNATSStreaming(url, clusterID, eventSourceName string, subject string, auth *eventbusdriver.Auth, logger *zap.SugaredLogger) *NATSStreaming {
	return &NATSStreaming{
		eventbusdriver.NewNATSStreaming(url, clusterID, auth, logger),
		eventSourceName,
		subject,
	}
}

func (n *NATSStreaming) Connect(clientID string) (SourceConnection, error) {
	conn, err := n.MakeConnection(clientID)
	if err != nil {
		return nil, err
	}

	return &NATSStreamingSourceConn{conn, n.eventSourceName, n.subject}, nil
}
