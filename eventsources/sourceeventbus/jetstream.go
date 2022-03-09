package sourceeventbus

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

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

func (n *Jetstream) Connect() (SourceConnection, error) {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	clientID := fmt.Sprintf("client-%s-%v", strings.ReplaceAll(n.hostname, ".", "_"), r1.Intn(1000))

	conn, err := n.MakeConnection(clientID)
	if err != nil {
		return nil, err
	}

	return &JetstreamSourceConn{conn, n.eventSourceName}, nil
}
