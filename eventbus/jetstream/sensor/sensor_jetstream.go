package sensor

import (
	"context"
	"fmt"
	"strings"
	"time"

	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	eventbusjetstreambase "github.com/argoproj/argo-events/eventbus/jetstream/base"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"encoding/json"

	"github.com/argoproj/argo-events/common"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	hashstructure "github.com/mitchellh/hashstructure/v2"
)

const (
	SensorNilError = "sensorSpec == nil??"
)

type SensorJetstream struct {
	*eventbusjetstreambase.Jetstream

	sensorName    string
	sensorSpec    *v1alpha1.Sensor
	keyValueStore nats.KeyValue
}

func NewSensorJetstream(url string, sensorSpec *v1alpha1.Sensor, streamConfig string, auth *eventbuscommon.Auth, logger *zap.SugaredLogger) (*SensorJetstream, error) {
	if sensorSpec == nil {
		errStr := SensorNilError
		logger.Errorf(errStr)
		return nil, fmt.Errorf(errStr)
	}

	baseJetstream, err := eventbusjetstreambase.NewJetstream(url, streamConfig, auth, logger)
	if err != nil {
		return nil, err
	}
	return &SensorJetstream{
		baseJetstream,
		sensorSpec.Name,
		sensorSpec,
		nil}, nil
}

func (stream *SensorJetstream) Initialize() error {
	err := stream.Init() // member of jetstreambase.Jetstream
	if err != nil {
		return err
	}

	// see if there's an existing one
	stream.keyValueStore, _ = stream.MgmtConnection.JSContext.KeyValue(stream.sensorName)
	if stream.keyValueStore == nil {
		// create Key/Value store for this Sensor (seems to be okay to call this if it already exists)
		stream.keyValueStore, err = stream.MgmtConnection.JSContext.CreateKeyValue(&nats.KeyValueConfig{Bucket: stream.sensorName})
		if err != nil {
			errStr := fmt.Sprintf("failed to Create Key/Value Store for sensor %s, err: %v", stream.sensorName, err)
			stream.Logger.Error(errStr)
			return err
		}
	} else {
		stream.Logger.Infof("found existing K/V store for sensor %s, using that", stream.sensorName)
	}
	stream.Logger.Infof("successfully created/located K/V store for sensor %s", stream.sensorName)

	// Here we can take the sensor specification and clean up the K/V store so as to remove any old
	// Triggers for this Sensor that no longer exist and any old Dependencies (and also Drain any corresponding Connections)
	err = stream.setStateToSpec(stream.sensorSpec)
	return err
}

func (stream *SensorJetstream) Connect(ctx context.Context, triggerName string, dependencyExpression string, deps []eventbuscommon.Dependency, atLeastOnce bool) (eventbuscommon.TriggerConnection, error) {
	conn, err := stream.MakeConnection()
	if err != nil {
		return nil, err
	}

	return NewJetstreamTriggerConn(conn, stream.sensorName, triggerName, dependencyExpression, deps)
}

// Update the K/V store to reflect the current Spec:
//  1. save the current spec, including list of triggers, list of dependencies and how they're defined, and trigger expressions
//  2. selectively purge dependencies from the K/V store if either the Trigger no longer exists,
//     the dependency definition has changed, or the trigger expression has changed
//  3. for each dependency purged, delete the associated consumer so no new data is sent there
func (stream *SensorJetstream) setStateToSpec(sensorSpec *v1alpha1.Sensor) error {
	log := stream.Logger
	if sensorSpec == nil {
		errStr := SensorNilError
		log.Error(errStr)
		return fmt.Errorf(errStr)
	}

	log.Infof("Comparing previous Spec stored in k/v store for sensor %s to new Spec", sensorSpec.Name)

	changedDeps, removedDeps, validDeps, err := stream.getChangedDeps(sensorSpec)
	if err != nil {
		return err
	}
	log.Infof("Comparison of previous dependencies definitions to current: changed=%v, removed=%v, still valid=%v", changedDeps, removedDeps, validDeps)

	changedTriggers, removedTriggers, validTriggers, err := stream.getChangedTriggers(sensorSpec) // this looks at the list of triggers as well as the dependency expression for each trigger
	if err != nil {
		return err
	}
	log.Infof("Comparison of previous trigger list to current: changed=%v, removed=%v, still valid=%v", changedTriggers, removedTriggers, validTriggers)

	// for all valid triggers, determine if changedDeps or removedDeps requires them to be deleted
	changedPlusRemovedDeps := make([]string, 0, len(changedDeps)+len(removedDeps))
	changedPlusRemovedDeps = append(changedPlusRemovedDeps, changedDeps...)
	changedPlusRemovedDeps = append(changedPlusRemovedDeps, removedDeps...)
	for _, triggerName := range validTriggers {
		_ = stream.purgeSelectedDepsForTrigger(triggerName, changedPlusRemovedDeps)
	}

	// for all changedTriggers (which includes modified and deleted), purge their dependencies
	changedPlusRemovedTriggers := make([]string, 0, len(changedTriggers)+len(removedTriggers))
	changedPlusRemovedTriggers = append(changedPlusRemovedTriggers, changedTriggers...)
	changedPlusRemovedTriggers = append(changedPlusRemovedTriggers, removedTriggers...)
	for _, triggerName := range changedPlusRemovedTriggers {
		_ = stream.purgeAllDepsForTrigger(triggerName)
	}

	// save new spec
	err = stream.saveSpec(sensorSpec, removedTriggers)
	if err != nil {
		return err
	}

	return nil
}

// purging dependency means both purging it from the K/V store as well as deleting the associated Consumer so no more messages are sent to it
func (stream *SensorJetstream) purgeDependency(triggerName string, depName string) error {
	// purge from Key/Value store first
	key := getDependencyKey(triggerName, depName)
	durableName := getDurableName(stream.sensorName, triggerName, depName)
	stream.Logger.Debugf("purging dependency, including 1) key %s from the K/V store, and 2) durable consumer %s", key, durableName)
	err := stream.keyValueStore.Delete(key)
	if err != nil && err != nats.ErrKeyNotFound { // sometimes we call this on a trigger/dependency combination not sure if it actually exists or not, so
		// don't need to worry about case of it not existing
		stream.Logger.Error(err)
		return err
	}
	// then delete consumer
	stream.Logger.Debugf("durable name for sensor='%s', trigger='%s', dep='%s': '%s'", stream.sensorName, triggerName, depName, durableName)

	_ = stream.MgmtConnection.JSContext.DeleteConsumer("default", durableName) // sometimes we call this on a trigger/dependency combination not sure if it actually exists or not, so
	// don't need to worry about case of it not existing

	return nil
}

func (stream *SensorJetstream) saveSpec(sensorSpec *v1alpha1.Sensor, removedTriggers []string) error {
	// remove the old triggers from the K/V store
	for _, trigger := range removedTriggers {
		key := getTriggerExpressionKey(trigger)
		err := stream.keyValueStore.Delete(key)
		if err != nil {
			errStr := fmt.Sprintf("error deleting key %s: %v", key, err)
			stream.Logger.Error(errStr)
			return fmt.Errorf(errStr)
		}
		stream.Logger.Debugf("successfully removed Trigger expression at key %s", key)
	}

	// save the dependency definitions
	depMap := make(DependencyDefinitionValue)
	for _, dep := range sensorSpec.Spec.Dependencies {
		hash, err := hashstructure.Hash(dep, hashstructure.FormatV2, nil)
		if err != nil {
			errStr := fmt.Sprintf("failed to hash dependency %+v", dep)
			stream.Logger.Errorf(errStr)
			err = fmt.Errorf(errStr)
			return err
		}
		depMap[dep.Name] = hash
	}
	err := stream.storeDependencyDefinitions(depMap)
	if err != nil {
		return err
	}

	// save the list of Triggers
	triggerList := make(TriggerValue, len(sensorSpec.Spec.Triggers))
	for i, trigger := range sensorSpec.Spec.Triggers {
		triggerList[i] = trigger.Template.Name
	}
	err = stream.storeTriggerList(triggerList)
	if err != nil {
		return err
	}

	// for each trigger, save its expression
	for _, trigger := range sensorSpec.Spec.Triggers {
		err := stream.storeTriggerExpression(trigger.Template.Name, trigger.Template.Conditions)
		if err != nil {
			return err
		}
	}

	return nil
}

func (stream *SensorJetstream) getChangedTriggers(sensorSpec *v1alpha1.Sensor) (changedTriggers []string, removedTriggers []string, validTriggers []string, err error) {
	if sensorSpec == nil {
		errStr := SensorNilError
		stream.Logger.Errorf(errStr)
		err = fmt.Errorf(errStr)
		return
	}

	mappedSpecTriggers := make(map[string]v1alpha1.Trigger, len(sensorSpec.Spec.Triggers))
	for _, trigger := range sensorSpec.Spec.Triggers {
		mappedSpecTriggers[trigger.Template.Name] = trigger
	}
	storedTriggers, err := stream.getTriggerList()
	if err != nil {
		return nil, nil, nil, err
	}
	for _, triggerName := range storedTriggers {
		currTrigger, found := mappedSpecTriggers[triggerName]
		if !found {
			removedTriggers = append(removedTriggers, triggerName)
		} else {
			// is the trigger expression the same or different?
			storedExpression, err := stream.getTriggerExpression(triggerName)
			if err != nil {
				return nil, nil, nil, err
			}
			if storedExpression == currTrigger.Template.Conditions {
				validTriggers = append(validTriggers, triggerName)
			} else {
				changedTriggers = append(changedTriggers, triggerName)
			}
		}
	}
	return changedTriggers, removedTriggers, validTriggers, nil
}

func (stream *SensorJetstream) getChangedDeps(sensorSpec *v1alpha1.Sensor) (changedDeps []string, removedDeps []string, validDeps []string, err error) {
	if sensorSpec == nil {
		errStr := SensorNilError
		stream.Logger.Errorf(errStr)
		err = fmt.Errorf(errStr)
		return
	}

	specDependencies := sensorSpec.Spec.Dependencies
	mappedSpecDependencies := make(map[string]v1alpha1.EventDependency, len(specDependencies))
	for _, dep := range specDependencies {
		mappedSpecDependencies[dep.Name] = dep
	}
	storedDependencies, err := stream.getDependencyDefinitions()
	if err != nil {
		return nil, nil, nil, err
	}
	for depName, hashedDep := range storedDependencies {
		currDep, found := mappedSpecDependencies[depName]
		if !found {
			removedDeps = append(removedDeps, depName)
		} else {
			// is the dependency definition the same or different?
			hash, err := hashstructure.Hash(currDep, hashstructure.FormatV2, nil)
			if err != nil {
				errStr := fmt.Sprintf("failed to hash dependency %+v", currDep)
				stream.Logger.Errorf(errStr)
				err = fmt.Errorf(errStr)
				return nil, nil, nil, err
			}
			if hash == hashedDep {
				validDeps = append(validDeps, depName)
			} else {
				changedDeps = append(changedDeps, depName)
			}
		}
	}

	return changedDeps, removedDeps, validDeps, nil
}

func (stream *SensorJetstream) getDependencyDefinitions() (DependencyDefinitionValue, error) {
	depDefs, err := stream.keyValueStore.Get(DependencyDefsKey)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return make(DependencyDefinitionValue), nil
		}
		errStr := fmt.Sprintf("error getting key %s: %v", DependencyDefsKey, err)
		stream.Logger.Error(errStr)
		return nil, fmt.Errorf(errStr)
	}
	stream.Logger.Debugf("Value of key %s: %s", DependencyDefsKey, string(depDefs.Value()))

	depDefMap := DependencyDefinitionValue{}
	err = json.Unmarshal(depDefs.Value(), &depDefMap)
	if err != nil {
		errStr := fmt.Sprintf("error unmarshalling value %s of key %s: %v", string(depDefs.Value()), DependencyDefsKey, err)
		stream.Logger.Error(errStr)
		return nil, fmt.Errorf(errStr)
	}

	return depDefMap, nil
}

func (stream *SensorJetstream) storeDependencyDefinitions(depDef DependencyDefinitionValue) error {
	bytes, err := json.Marshal(depDef)
	if err != nil {
		errStr := fmt.Sprintf("error marshalling %+v: %v", depDef, err)
		stream.Logger.Error(errStr)
		return fmt.Errorf(errStr)
	}
	_, err = stream.keyValueStore.Put(DependencyDefsKey, bytes)
	if err != nil {
		errStr := fmt.Sprintf("error storing %s under key %s: %v", string(bytes), DependencyDefsKey, err)
		stream.Logger.Error(errStr)
		return fmt.Errorf(errStr)
	}
	stream.Logger.Debugf("successfully stored dependency definition under key %s: %s", DependencyDefsKey, string(bytes))
	return nil
}

func (stream *SensorJetstream) getTriggerList() (TriggerValue, error) {
	triggerListJson, err := stream.keyValueStore.Get(TriggersKey)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return make(TriggerValue, 0), nil
		}
		errStr := fmt.Sprintf("error getting key %s: %v", TriggersKey, err)
		stream.Logger.Error(errStr)
		return nil, fmt.Errorf(errStr)
	}
	stream.Logger.Debugf("Value of key %s: %s", TriggersKey, string(triggerListJson.Value()))

	triggerList := TriggerValue{}
	err = json.Unmarshal(triggerListJson.Value(), &triggerList)
	if err != nil {
		errStr := fmt.Sprintf("error unmarshalling value %s of key %s: %v", string(triggerListJson.Value()), TriggersKey, err)
		stream.Logger.Error(errStr)
		return nil, fmt.Errorf(errStr)
	}

	return triggerList, nil
}

func (stream *SensorJetstream) storeTriggerList(triggerList TriggerValue) error {
	bytes, err := json.Marshal(triggerList)
	if err != nil {
		errStr := fmt.Sprintf("error marshalling %+v: %v", triggerList, err)
		stream.Logger.Error(errStr)
		return fmt.Errorf(errStr)
	}
	_, err = stream.keyValueStore.Put(TriggersKey, bytes)
	if err != nil {
		errStr := fmt.Sprintf("error storing %s under key %s: %v", string(bytes), TriggersKey, err)
		stream.Logger.Error(errStr)
		return fmt.Errorf(errStr)
	}
	stream.Logger.Debugf("successfully stored trigger list under key %s: %s", TriggersKey, string(bytes))
	return nil
}

func (stream *SensorJetstream) getTriggerExpression(triggerName string) (string, error) {
	key := getTriggerExpressionKey(triggerName)
	expr, err := stream.keyValueStore.Get(key)
	if err != nil {
		if err == nats.ErrKeyNotFound {
			return "", nil
		}
		errStr := fmt.Sprintf("error getting key %s: %v", key, err)
		stream.Logger.Error(errStr)
		return "", fmt.Errorf(errStr)
	}
	stream.Logger.Debugf("Value of key %s: %s", key, string(expr.Value()))

	return string(expr.Value()), nil
}

func (stream *SensorJetstream) storeTriggerExpression(triggerName string, conditionExpression string) error {
	key := getTriggerExpressionKey(triggerName)
	_, err := stream.keyValueStore.PutString(key, conditionExpression)
	if err != nil {
		errStr := fmt.Sprintf("error storing %s under key %s: %v", conditionExpression, key, err)
		stream.Logger.Error(errStr)
		return fmt.Errorf(errStr)
	}
	stream.Logger.Debugf("successfully stored trigger expression under key %s: %s", key, conditionExpression)
	return nil
}

func (stream *SensorJetstream) purgeSelectedDepsForTrigger(triggerName string, deps []string) error {
	stream.Logger.Debugf("purging selected dependencies %v for trigger %s", deps, triggerName)
	for _, dep := range deps {
		err := stream.purgeDependency(triggerName, dep) // this will attempt a delete even if no such key exists for a particular trigger, but that's okay
		if err != nil {
			return err
		}
	}
	return nil
}

func (stream *SensorJetstream) purgeAllDepsForTrigger(triggerName string) error {
	stream.Logger.Debugf("purging all dependencies for trigger %s", triggerName)
	// use the stored trigger expression to determine which dependencies need to be purged
	storedExpression, err := stream.getTriggerExpression(triggerName)
	if err != nil {
		return err
	}

	// get the individual dependencies by removing the special characters
	modExpr := strings.ReplaceAll(storedExpression, "&&", " ")
	modExpr = strings.ReplaceAll(modExpr, "||", " ")
	modExpr = strings.ReplaceAll(modExpr, "(", " ")
	modExpr = strings.ReplaceAll(modExpr, ")", " ")
	deps := strings.FieldsFunc(modExpr, func(r rune) bool { return r == ' ' })

	for _, dep := range deps {
		err := stream.purgeDependency(triggerName, dep)
		if err != nil {
			return err
		}
	}
	return nil
}

// ////////////////////////////////////////////////////////////////////////////////////////////////////
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

// ////////////////////////////////////////////////////////////////////////////////////////////////////
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
