package sensoreventbus

import (
	"fmt"
	"time"

	"math/rand"

	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
)

type Jetstream struct {
	*eventbusdriver.Jetstream

	sensorName    string
	keyValueStore *nats.KeyValue
}

func NewJetstream(url string, sensorName string, auth *eventbusdriver.Auth, logger *zap.SugaredLogger) Driver {
	return &Jetstream{
		eventbusdriver.NewJetstream(url, auth, logger),
		sensorName,
		nil,
	}
}

func (stream *Jetstream) Connect(triggerName string, dependencyExpression string, deps []Dependency) (TriggerConnection, error) {
	// Generate clientID with hash code
	hashKey := fmt.Sprintf("%s-%s", stream.sensorName, triggerName)

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	hashVal := common.Hasher(hashKey)
	clientID := fmt.Sprintf("client-%v-%v", hashVal, r1.Intn(100))

	conn, err := stream.MakeConnection(clientID)
	if err != nil {
		return nil, err
	}

	return &JetstreamTriggerConn{conn, stream.sensorName, triggerName, nil, dependencyExpression, deps}, nil

}

// todo: when we subscribe later we can specify durable name there (look at SubOpt in js.go)
// maybe also use AckAll()? need to look for all SubOpt
