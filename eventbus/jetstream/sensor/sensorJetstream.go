package sensor

import (
	"fmt"
	"time"

	"errors"
	"math/rand"

	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	eventbusjetstreambase "github.com/argoproj/argo-events/eventbus/jetstream/base"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"encoding/json"

	"github.com/argoproj/argo-events/common"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type SensorJetstream struct {
	*eventbusjetstreambase.Jetstream

	sensorName    string
	keyValueStore nats.KeyValue
}

func NewSensorJetstream(url string, sensorSpec *v1alpha1.Sensor, auth *eventbuscommon.Auth, logger *zap.SugaredLogger) (*SensorJetstream, error) {
	if sensorSpec == nil {
		errStr := fmt.Sprintf("sensorSpec == nil??")
		logger.Errorf(errStr)
		return nil, errors.New(errStr)
	}

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
func (stream *SensorJetstream) setStateToSpec(sensorSpec *v1alpha1.Sensor) error {
	if sensorSpec == nil {
		errStr := fmt.Sprintf("sensorSpec == nil??")
		stream.Logger.Error(errStr)
		return errors.New(errStr)
	}

	changedDeps, removedDeps, validDeps, err := stream.getChangedDeps(sensorSpec)
	changedTriggers, removedTriggers, validTriggers := stream.getChangedTriggers(sensorSpec)

	// for all valid triggers, determine if changedDeps or removedDeps requires them to be deleted
	for _, triggerName := range validTriggers {
		stream.purgeSelectedDepsForTrigger(triggerName, changedDeps, removedDeps)
	}

	// for all changedTriggers (which includes modified and deleted), purge their dependencies
	for _, triggerName := range changedTriggers {
		stream.purgeAllDepsForTrigger(triggerName)
	}
	for _, triggerName := range removedTriggers {
		stream.purgeAllDepsForTrigger(triggerName)
	}

	// save new spec
	stream.saveSpec(sensorSpec, removedTriggers)

	return nil
}

// purging dependency means both purging it from the K/V store as well as deleting the associated Consumer so no more messages are sent to it
func (stream *SensorJetstream) purgeDependency(triggerName string, depName string) error {
	// purge from Key/Value store first
	key := getDependencyKey(triggerName, depName)
	stream.Logger.Debugf("clearing key %s from the K/V store", key)
	err := stream.keyValueStore.Delete(key)
	if err != nil && err != nats.ErrKeyNotFound {
		stream.Logger.Error(err)
		return err
	}

	// then delete consumer
	durableName := getDurableName(stream.sensorName, triggerName, depName)

}

func (stream *SensorJetstream) getChangedDeps(sensorSpec *v1alpha1.Sensor) (changedDeps []string, removedDeps []string, validDeps []string, err error) {
	if sensorSpec == nil {
		errStr := fmt.Sprintf("sensorSpec == nil??")
		stream.Logger.Errorf(errStr)
		err = errors.New(errStr)
		return
	}

	specDependencies := sensorSpec.Spec.Dependencies
	storedDependencies := stream.getDependencyDefinitions()

}

func (stream *SensorJetstream) getDependencyDefinitions() (DependencyDefinitionValue, error) {
	depDefs, err := stream.keyValueStore.Get(DependencyDefsKey)
	if err != nil {
		errStr := fmt.Sprintf("error getting key %s: %v", DependencyDefsKey, err)
		stream.Logger.Error(errStr)
		return nil, errors.New(errStr)
	}
	stream.Logger.Debugf("Value of key %s: %s", DependencyDefsKey, string(depDefs.Value()))

	depDefMap := DependencyDefinitionValue{}
	err = json.Unmarshal(depDefs.Value(), &depDefMap)
	if err != nil {
		errStr := fmt.Sprintf("error unmarshalling value %s of key %s: %v", string(depDefs.Value()), DependencyDefsKey, err)
		stream.Logger.Error(errStr)
		return nil, errors.New(errStr)
	}

	return depDefMap, nil
}

func (stream *SensorJetstream) storeDependencyDefinitions(depDef DependencyDefinitionValue) error {

}

//////////////////////////////////////////////////////////////////////////////////////////////////////
// These are the Keys and methods to derive Keys for our K/V store
var (
	TriggersKey       = "Triggers"
	DependencyDefsKey = "Deps"
)

func getDependencyKey(triggerName string, depName string) string {
	return fmt.Sprintf("%s/%s", triggerName, depName)
}

func getTriggerExpressionKey(triggerName string) string {
	return fmt.Sprintf("%s/Expression", triggerName)
}

//////////////////////////////////////////////////////////////////////////////////////////////////////
// These are the structs representing Values in our K/V store
type DependencyDefinitionValue map[string]uint64 // value for DependencyDefsKey
type TriggerValue []string                       // value for TriggersKey

// value for getDependencyKey()
type MsgInfo struct {
	StreamSeq   uint64
	ConsumerSeq uint64
	Timestamp   time.Time
	Event       *cloudevents.Event
}

func getDurableName(sensorName string, triggerName string, depName string) string {
	hashKey := fmt.Sprintf("%s-%s-%s", sensorName, triggerName, depName)
	hashVal := common.Hasher(hashKey)
	return fmt.Sprintf("group-%s", hashVal)
}
