package sensor

import (
	"fmt"
	"time"

	"math/rand"

	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	eventbusjetstreambase "github.com/argoproj/argo-events/eventbus/jetstream/base"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/common"
)

type SensorJetstream struct {
	*eventbusjetstreambase.Jetstream

	sensorName    string
	keyValueStore nats.KeyValue
}

func NewSensorJetstream(url string, sensorSpec *v1alpha1.Sensor, auth *eventbuscommon.Auth, logger *zap.SugaredLogger) (*SensorJetstream, error) {

	baseJetstream, err := eventbusjetstreambase.NewJetstream(url, auth, logger)
	if err != nil {
		return nil, err
	}

	// todo: determine if we can also call this if it already exists
	kvStore, err := baseJetstream.MgmtConnection.JSContext.CreateKeyValue(&nats.KeyValueConfig{Bucket: sensorSpec.Name})
	if err != nil {
		errStr := fmt.Sprintf("failed to Create Key/Value Store for sensor %s, err: %v", sensorSpec.Name, err)
		logger.Error(errStr)
		return nil, err
	}
	logger.Infof("successfully created K/V store for sensor %s (if it doesn't already exist)", sensorSpec.Name)

	// todo: Here we can take the sensor specification and clean up the K/V store so as to remove any old
	// Triggers for this Sensor that no longer exist and any old Dependencies (and also Drain any corresponding Connections)

	return &SensorJetstream{
		baseJetstream,
		sensorSpec.Name,
		kvStore,
	}, nil
}

func (stream *SensorJetstream) Connect(triggerName string, dependencyExpression string, deps []eventbuscommon.Dependency) (eventbuscommon.TriggerConnection, error) {
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

	return NewJetstreamTriggerConn(conn, stream.sensorName, triggerName, dependencyExpression, deps)
}

func getDependencyKey(triggerName string, depName string) string {
	return fmt.Sprintf("%s/%s", triggerName, depName)
}
