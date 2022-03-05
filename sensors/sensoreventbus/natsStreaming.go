package sensoreventbus

import (
	"fmt"
	"rand"
	"time"

	"github.com/argoproj/argo-events/common"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	"go.uber.org/zap"
)

type NATSStreaming struct {
	*eventbusdriver.NATStreaming
	sensorName string
}

func NewNATSStreaming(url, clusterID, sensorName string, auth *eventbusdriver.Auth, logger *zap.SugaredLogger) *NATSStreaming {
	return &NATSStreaming{
		eventbusdriver.NewNATSStreaming(url, clusterID, auth, logger),
		sensorName,
	}
}

func (n *NATSStreaming) Connect(triggerName string, dependencyExpression string, deps []eventbusdriver.Dependency) (eventbusdriver.TriggerConnection, error) {
	// Generate clientID with hash code
	hashKey := fmt.Sprintf("%s-%s-%s", n.sensorName, triggerName, dependencyExpression)

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	hashVal := common.Hasher(hashKey)
	clientID := fmt.Sprintf("client-%v-%v", hashVal, r1.Intn(100))

	conn, err := n.MakeConnection(clientID)
	if err != nil {
		return nil, err
	}

	return &NATSStreamingTriggerConn{conn, n.sensorName, triggerName}, nil
}
