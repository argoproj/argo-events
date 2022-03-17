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

func NewSensorJetstream(url string, sensorSpec v1alpha1.Sensor, auth *eventbuscommon.Auth, logger *zap.SugaredLogger) (*SensorJetstream, error) {

	baseJetstream, err := eventbusjetstreambase.NewJetstream(url, auth, logger)
	if err != nil {
		return nil, err
	}

	// create Key/Value store for this Sensor (seems to be okay to call this if it already exists)
	kvStore, err := baseJetstream.MgmtConnection.JSContext.CreateKeyValue(&nats.KeyValueConfig{Bucket: sensorSpec.Name})
	if err != nil {
		errStr := fmt.Sprintf("failed to Create Key/Value Store for sensor %s, err: %v", sensorSpec.Name, err)
		logger.Error(errStr)
		return nil, err
	}
	logger.Infof("successfully created K/V store for sensor %s (if it doesn't already exist)", sensorSpec.Name)

	jetstream := &SensorJetstream{
		baseJetstream,
		sensorSpec.Name,
		kvStore,
	}

	// todo: Here we can take the sensor specification and clean up the K/V store so as to remove any old
	// Triggers for this Sensor that no longer exist and any old Dependencies (and also Drain any corresponding Connections)
	err = jetstream.setStateToSpec(sensorSpec)
	if err != nil {
		return jetstream, err
	}

	return jetstream, nil
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

// Update the K/V store to reflect the current Spec:
// 1. save the current spec, including list of triggers, list of dependencies and how they're defined, and trigger expressions
// 2. selectively purge dependencies from the K/V store if either the Trigger no longer exists,
//    the dependency definition has changed, or the trigger expression has changed
// 3. for each dependency purged, delete the associated consumer so no new data is sent there
func (stream *SensorJetstream) setStateToSpec(sensorSpec v1alpha1.Sensor) error {
	return nil
}

var (
	TriggersKey     = "Triggers"
	DependenciesKey = "Deps"
)

func getDependencyKey(triggerName string, depName string) string {
	return fmt.Sprintf("%s/%s", triggerName, depName)
}

func getTriggerExpressionKey(triggerName string) string {
	return fmt.Sprintf("%s/Expression", triggerName)
}

func getDurableName(sensorName string, triggerName string, depName string) string {
	hashKey := fmt.Sprintf("%s-%s-%s", sensorName, triggerName, depName)
	hashVal := common.Hasher(hashKey)
	return fmt.Sprintf("group-%s", hashVal)
}
