package sensor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Knetic/govaluate"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"

	"encoding/json"

	"github.com/argoproj/argo-events/common"
	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	jetstreambase "github.com/argoproj/argo-events/eventbus/jetstream/base"
	nats "github.com/nats-io/nats.go"
)

type JetstreamTriggerConn struct {
	*jetstreambase.JetstreamConnection
	sensorName           string
	triggerName          string
	keyValueStore        nats.KeyValue
	dependencyExpression string
	requiresANDLogic     bool
	evaluableExpression  *govaluate.EvaluableExpression
	deps                 []eventbuscommon.Dependency
	sourceDepMap         map[string][]string // maps EventSource and EventName to dependency name
}

func NewJetstreamTriggerConn(conn *jetstreambase.JetstreamConnection,
	sensorName string,
	triggerName string,
	dependencyExpression string,
	deps []eventbuscommon.Dependency) (*JetstreamTriggerConn, error) {
	var err error

	sourceDepMap := make(map[string][]string)
	for _, d := range deps {
		key := d.EventSourceName + "__" + d.EventName
		_, found := sourceDepMap[key]
		if !found {
			sourceDepMap[key] = make([]string, 0)
		}
		sourceDepMap[key] = append(sourceDepMap[key], d.Name)
	}

	connection := &JetstreamTriggerConn{
		JetstreamConnection:  conn,
		sensorName:           sensorName,
		triggerName:          triggerName,
		dependencyExpression: dependencyExpression,
		requiresANDLogic:     strings.Contains(dependencyExpression, "&"),
		deps:                 deps,
		sourceDepMap:         sourceDepMap}
	connection.Logger = connection.Logger.With("triggerName", connection.triggerName).With("clientID", connection.ClientID())

	connection.evaluableExpression, err = govaluate.NewEvaluableExpression(strings.ReplaceAll(dependencyExpression, "-", "\\-"))
	if err != nil {
		return nil, err
		errStr := fmt.Sprintf("failed to evaluate expression %s: %v", dependencyExpression, err)
		connection.Logger.Error(errStr)
		return nil, errors.New(errStr)
	}

	connection.keyValueStore, err = conn.JSContext.KeyValue(sensorName)
	if err != nil {
		errStr := fmt.Sprintf("failed to get K/V store for sensor %s: %v", sensorName, err)
		connection.Logger.Error(errStr)
		return nil, errors.New(errStr)
	}

	connection.Logger.Infof("Successfully located K/V store for sensor %s", sensorName)
	return connection, nil
}

func (conn *JetstreamTriggerConn) Subscribe(ctx context.Context,
	closeCh <-chan struct{},
	resetConditionsCh <-chan struct{},
	lastResetTime time.Time,
	transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
	filter func(string, cloudevents.Event) bool,
	action func(map[string]cloudevents.Event),
	defaultSubject *string) error {
	log := conn.Logger
	// derive subjects that we'll subscribe with using the dependencies passed in
	subjects := make(map[string]eventbuscommon.Dependency)
	for _, dep := range conn.deps {
		subjects[fmt.Sprintf("default.%s.%s", dep.EventSourceName, dep.EventName)] = dep
	}

	err := conn.CleanUpOnStart()
	if err != nil {
		return err
	}

	if !lastResetTime.IsZero() {
		conn.clearAllDependencies(&lastResetTime)
	}

	ch := make(chan *nats.Msg, 64) // todo: 64 is random - make a constant? any concerns about it not being big enough?
	wg := sync.WaitGroup{}
	processMsgsCloseCh := make(chan struct{})

	subscriptions := make([]*nats.Subscription, len(subjects))
	subscriptionIndex := 0

	// start the goroutines that will listen to the individual subscriptions
	for subject, dependency := range subjects {
		// set durable name separately for each subscription
		hashKey := fmt.Sprintf("%s-%s-%s-%s", conn.sensorName, conn.triggerName, dependency.EventSourceName, dependency.EventName)
		hashVal := common.Hasher(hashKey)
		durableName := fmt.Sprintf("group-%s", hashVal)

		log.Infof("Subscribing to subject %s with durable name %s", subject, durableName)
		subscriptions[subscriptionIndex], err = conn.JSContext.PullSubscribe(subject, durableName, nats.AckExplicit()) // todo: what other subscription options here?
		if err != nil {
			log.Errorf("Failed to subscribe to subject %s using group %s: %v", subject, durableName, err)
			continue
		}
		go conn.pullSubscribe(subscriptions[subscriptionIndex], ch, processMsgsCloseCh, wg)
		wg.Add(1)

		subscriptionIndex++
	}

	// create a single goroutine which which handle receiving messages to ensure that all of the processing is occurring on that
	// one goroutine and we don't need to worry about race conditions
	go conn.processMsgs(ch, processMsgsCloseCh, resetConditionsCh, transform, filter, action, wg)
	wg.Add(1)

	for {
		select {
		case <-ctx.Done():
			log.Info("exiting, closing connection...")
			processMsgsCloseCh <- struct{}{}
			wg.Wait()
			conn.NATSConn.Close()
			return nil
		case <-closeCh:
			log.Info("closing connection...")
			processMsgsCloseCh <- struct{}{}
			wg.Wait()
			conn.NATSConn.Close()
			return nil
		}
	}

}

func (conn *JetstreamTriggerConn) pullSubscribe(
	subscription *nats.Subscription,
	msgChannel chan<- *nats.Msg,
	closeCh <-chan struct{},
	wg sync.WaitGroup) {
	for {
		// call Fetch with timeout
		msgs, err := subscription.Fetch(1, nats.MaxWait(time.Second*1))
		if err != nil && !errors.Is(err, nats.ErrTimeout) {
			conn.Logger.Errorf("failed to fetch messages for subscription %+v, %v", subscription, err)
		}

		// then check to see if closeCh has anything; if so, wg.Done() and exit
		if len(closeCh) != 0 {
			wg.Done()
			conn.Logger.Infof("exiting pullSubscribe() for subscription %+v", subscription)
			return
		}

		// then push the msgs to the channel which will consume them
		for _, msg := range msgs {
			msgChannel <- msg
		}
	}
}

func (conn *JetstreamTriggerConn) processMsgs(
	receiveChannel <-chan *nats.Msg,
	closeCh <-chan struct{},
	resetConditionsCh <-chan struct{},
	transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
	filter func(string, cloudevents.Event) bool,
	action func(map[string]cloudevents.Event),
	wg sync.WaitGroup) {

	defer wg.Done()

	for {
		select {
		case msg := <-receiveChannel:
			conn.processMsg(msg, transform, filter, action)
		case <-resetConditionsCh:
			conn.Logger.Info("reset conditions")
			conn.clearAllDependencies(nil)
		case <-closeCh:
			conn.Logger.Info("shutting down processMsgs routine")
			wg.Done()
			return
		}
	}
}

func (conn *JetstreamTriggerConn) processMsg(
	m *nats.Msg,
	transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
	filter func(string, cloudevents.Event) bool,
	action func(map[string]cloudevents.Event)) {

	defer m.Ack() // todo: how do we do Exactly once delivery?:
	// Documentation says: "Consumers can be 100% sure a message was correctly processed by "
	// requesting the server Acknowledge having received your acknowledgement by setting a reply
	// subject on the Ack. If you receive this response you will never receive that message again."
	log := conn.Logger

	var event *cloudevents.Event
	if err := json.Unmarshal(m.Data, &event); err != nil {
		log.Errorf("Failed to convert to a cloudevent, discarding it... err: %v", err)
		return
	}

	// get all dependencies for this Trigger that match
	depNames, err := conn.getDependencyNames(event.Source(), event.Subject())
	if err != nil || len(depNames) == 0 {
		log.Errorf("Failed to get the dependency names, discarding it... err: %v", err)
		return
	}

	log.Debugf("New incoming Event Source Message, dependency names=%s", depNames)

	for _, depName := range depNames {
		conn.processDependency(m, event, depName, transform, filter, action)
	}
}

func (conn *JetstreamTriggerConn) processDependency(
	m *nats.Msg,
	event *cloudevents.Event,
	depName string,
	transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
	filter func(string, cloudevents.Event) bool,
	action func(map[string]cloudevents.Event)) {

	log := conn.Logger
	event, err := transform(depName, *event)
	if err != nil {
		log.Errorw("failed to apply event transformation, ", err)
		return
	}

	if !filter(depName, *event) {
		// message not interested
		log.Debugf("not interested in dependency %s (didn't pass filter)", depName)
		return
	}

	if !conn.requiresANDLogic {
		// this is the simple case: we can just perform the trigger
		messages := make(map[string]cloudevents.Event)
		messages[depName] = *event
		log.Infof("Triggering actions after receiving dependency %s", depName)

		action(messages)
	} else {
		// check Dependency expression (need to retrieve previous dependencies from Key/Value store)

		prevMsgs, err := conn.getSavedDependencies()
		if err != nil {
			return
		}

		// populate 'parameters' map to indicate which dependencies have been received and which haven't
		parameters := make(map[string]interface{}, len(conn.deps))
		for _, dep := range conn.deps {
			parameters[dep.Name] = false
		}
		for prevDep, _ := range prevMsgs {
			parameters[prevDep] = true
		}
		parameters[depName] = true
		log.Debugf("Current state of dependencies: %v", parameters)

		// evaluate the filter expression
		result, err := conn.evaluableExpression.Evaluate(parameters)
		if err != nil {
			errStr := fmt.Sprintf("failed to evaluate dependency expression: %v", err)
			log.Error(errStr)
			return
		}

		// if expression is true, trigger and clear the K/V store
		// else save the new message in the K/V store
		if result == true {
			log.Debugf("dependency expression successfully evaluated to true: %s", conn.dependencyExpression)

			messages := make(map[string]cloudevents.Event, len(prevMsgs)+1)
			for prevDep, msgInfo := range prevMsgs {
				messages[prevDep] = *msgInfo.Event
			}
			messages[depName] = *event
			log.Infof("Triggering actions after receiving dependency %s", depName)

			action(messages)

			err = conn.clearAllDependencies(nil)
			if err != nil {
				return
			}
		} else {
			log.Debugf("dependency expression false: %s", conn.dependencyExpression)
			msgMetadata, err := m.Metadata()
			if err != nil {
				errStr := fmt.Sprintf("message %+v is not a jetstream message???: %v", m, err)
				log.Error(errStr)
				return
			}
			err = conn.saveDependency(depName,
				MsgInfo{
					StreamSeq:   msgMetadata.Sequence.Stream,
					ConsumerSeq: msgMetadata.Sequence.Consumer,
					Timestamp:   msgMetadata.Timestamp,
					Event:       event})
		}

	}
}

func (conn *JetstreamTriggerConn) getSavedDependencies() (map[string]MsgInfo, error) {
	// dependencies are formatted "<Sensor>/<Trigger>/<Dependency>""
	prevMsgs := make(map[string]MsgInfo)

	// for each dependency that's in our dependency expression, look for it:
	for _, dep := range conn.deps {
		msgInfo, found, err := conn.getSavedDependency(dep.Name)
		if err != nil {
			return prevMsgs, err
		}
		if found {
			prevMsgs[dep.Name] = msgInfo
		}
	}

	return prevMsgs, nil
}

func (conn *JetstreamTriggerConn) getSavedDependency(depName string) (msg MsgInfo, found bool, err error) {
	key := getDependencyKey(conn.triggerName, depName)
	entry, err := conn.keyValueStore.Get(key)
	if err == nil {
		if entry != nil {
			var msgInfo MsgInfo
			err := json.Unmarshal(entry.Value(), &msgInfo)
			if err != nil {
				errStr := fmt.Sprintf("error unmarshalling value %s for key %s: %v", string(entry.Value()), key, err)
				conn.Logger.Error(errStr)
				return MsgInfo{}, true, errors.New(errStr)
			}
			return msgInfo, true, nil
		}
	} else if err != nats.ErrKeyNotFound {
		return MsgInfo{}, false, err
	}

	return MsgInfo{}, false, nil
}

func (conn *JetstreamTriggerConn) saveDependency(depName string, msgInfo MsgInfo) error {
	log := conn.Logger
	jsonEncodedMsg, err := json.Marshal(msgInfo)
	if err != nil {
		errorStr := fmt.Sprintf("failed to convert msgInfo struct into JSON: %+v", msgInfo)
		log.Error(errorStr)
		return errors.New(errorStr)
	}
	key := getDependencyKey(conn.triggerName, depName)

	_, err = conn.keyValueStore.Put(key, jsonEncodedMsg)
	if err != nil {
		errorStr := fmt.Sprintf("failed to store dependency under key %s, value:%s: %+v", key, jsonEncodedMsg, err)
		log.Error(errorStr)
		return errors.New(errorStr)
	}

	return nil

}

func (conn *JetstreamTriggerConn) clearAllDependencies(beforeTimeOpt *time.Time) error {
	for _, dep := range conn.deps {
		if beforeTimeOpt != nil && !beforeTimeOpt.IsZero() {
			err := conn.clearDependencyIfExistsBeforeTime(dep.Name, *beforeTimeOpt)
			if err != nil {
				return err
			}
		} else {
			err := conn.clearDependencyIfExists(dep.Name)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (conn *JetstreamTriggerConn) clearDependencyIfExistsBeforeTime(depName string, beforeTime time.Time) error {
	key := getDependencyKey(conn.triggerName, depName)

	// first get the value (if it exists) to determine if it occurred before or after the time in question
	msgInfo, found, err := conn.getSavedDependency(depName)
	if err != nil {
		return err
	}
	if found {
		// determine if the dependency is from before the time in question
		if msgInfo.Timestamp.Before(beforeTime) {
			conn.Logger.Debugf("clearing key %s from the K/V store since its message time %+v occurred before %+v; MsgInfo:%+v",
				key, msgInfo.Timestamp.Local(), beforeTime.Local(), msgInfo)
			err := conn.keyValueStore.Delete(key)
			if err != nil && err != nats.ErrKeyNotFound {
				conn.Logger.Error(err)
				return err
			}
		}
	}

	return nil
}

func (conn *JetstreamTriggerConn) clearDependencyIfExists(depName string) error {
	key := getDependencyKey(conn.triggerName, depName)
	conn.Logger.Debugf("clearing key %s from the K/V store", key)
	err := conn.keyValueStore.Delete(key)
	if err != nil && err != nats.ErrKeyNotFound {
		conn.Logger.Error(err)
		return err
	}
	return nil
}

func (conn *JetstreamTriggerConn) getDependencyNames(eventSourceName, eventName string) ([]string, error) {
	deps, found := conn.sourceDepMap[eventSourceName+"__"+eventName]
	if !found {
		errStr := fmt.Sprintf("incoming event source and event not associated with any dependencies, event source=%s, event=%s",
			eventSourceName, eventName)
		conn.Logger.Error(errStr)
		return nil, errors.New(errStr)
	}

	return deps, nil
}

func (conn *JetstreamTriggerConn) CleanUpOnStart() error {
	// look in K/V store for Trigger expressions that have changed

	// for each Trigger that no longer exists, need to handle:
	// - messages sent for that Trigger that are in the K/V store
	// - messages sent to that Trigger that never reached it and are waiting in the eventbus (need to make a new connection and Drain())

	return nil
}

// this is what we'll store in our K/V store for each dependency (JSON encoded)
// key: "<Sensor>/<Trigger>/<Dependency>""
type MsgInfo struct {
	StreamSeq   uint64
	ConsumerSeq uint64
	Timestamp   time.Time
	Event       *cloudevents.Event
}
