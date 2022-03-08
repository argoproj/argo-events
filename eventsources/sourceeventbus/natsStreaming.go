package sourceeventbus

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	"go.uber.org/zap"
)

type NATSStreaming struct {
	*eventbusdriver.NATStreaming
	eventSourceName string
}

func NewNATSStreaming(url, clusterID, eventSourceName string, auth *eventbusdriver.Auth, logger *zap.SugaredLogger) *NATSStreaming {
	return &NATSStreaming{
		eventbusdriver.NewNATSStreaming(url, clusterID, auth, logger),
		eventSourceName,
	}
}

func (n *NATSStreaming) Connect() (SourceConnection, error) {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	clientID := fmt.Sprintf("client-%s-%v", strings.ReplaceAll(hostname, ".", "_"), r1.Intn(1000))

	conn, err := n.MakeConnection(clientID)
	if err != nil {
		return nil, err
	}

	return &NATSStreamingSourceConnection{conn, n.eventSourceName}, nil
}
