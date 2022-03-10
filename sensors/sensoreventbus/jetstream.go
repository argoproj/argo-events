package sensoreventbus

import (
	"fmt"
	"time"

	"math/rand"

	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
)

type Jetstream struct {
	*eventbusdriver.Jetstream

	sensorName    string
	keyValueStore *nats.KeyValue
}

func NewJetstream(url string, sensorSpec *v1alpha1.Sensor, auth *eventbusdriver.Auth, logger *zap.SugaredLogger) Driver {

	// todo: Here we can take the sensor specification and clean up the K/V store so as to remove any old
	// Triggers for this Sensor that no longer exist and any old Dependencies

	return &Jetstream{
		eventbusdriver.NewJetstream(url, auth, logger),
		sensorSpec.Name,
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

	return NewJetstreamTriggerConn(conn, stream.sensorName, triggerName, nil, dependencyExpression, deps), nil

}
