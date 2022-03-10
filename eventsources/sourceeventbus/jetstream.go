package sourceeventbus

import (
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	"go.uber.org/zap"
)

type Jetstream struct {
	*eventbusdriver.Jetstream
	eventSourceName string
	hostname        string
}

func NewJetstream(url, clusterID, eventSourceName string, hostname string, auth *eventbusdriver.Auth, logger *zap.SugaredLogger) *Jetstream {
	return &Jetstream{
		eventbusdriver.NewJetstream(url, auth, logger),
		eventSourceName,
		hostname,
	}

}

func (n *Jetstream) Connect(clientID string) (SourceConnection, error) {
	conn, err := n.MakeConnection(clientID)
	if err != nil {
		return nil, err
	}

	return &JetstreamSourceConn{conn, n.eventSourceName}, nil
}
