package sensor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Knetic/govaluate"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	nats "github.com/nats-io/nats.go"

	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	jetstreambase "github.com/argoproj/argo-events/eventbus/jetstream/base"
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
	recentMsgsByID       map[string]*msg     // prevent re-processing the same message as before (map of msg ID to time)
	recentMsgsByTime     []*msg
}

type msg struct {
	time  int64
	msgID string
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
		sourceDepMap:         sourceDepMap,
		recentMsgsByID:       make(map[string]*msg),
		recentMsgsByTime:     make([]*msg, 0)}
	connection.Logger = connection.Logger.With("triggerName", connection.triggerName)

	connection.evaluableExpression, err = govaluate.NewEvaluableExpression(strings.ReplaceAll(dependencyExpression, "-", "\\-"))
	if err != nil {
		errStr := fmt.Sprintf("failed to evaluate expression %s: %v", dependencyExpression, err)
		connection.Logger.Error(errStr)
		return nil, fmt.Errorf(errStr)
	}

	connection.keyValueStore, err = conn.JSContext.KeyValue(sensorName)
	if err != nil {
		errStr := fmt.Sprintf("failed to get K/V store for sensor %s: %v", sensorName, err)
		connection.Logger.Error(errStr)
		return nil, fmt.Errorf(errStr)
	}

	connection.Logger.Infof("Successfully located K/V store for sensor %s", sensorName)
	return connection, nil
}

func (conn *JetstreamTriggerConn) String() string {
	if conn == nil {
		return ""
	}
	return fmt.Sprintf("JetstreamTriggerConn{Sensor:%s,Trigger:%s}", conn.sensorName, conn.triggerName)
}

func (conn *JetstreamTriggerConn) Subscribe(ctx context.Context,
	closeCh <-chan struct{},
	resetConditionsCh <-chan struct{},
	lastResetTime time.Time,
	transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
	filter func(string, cloudevents.Event) bool,
	action func(map[string]cloudevents.Event),
	defaultSubject *string) error {
	if conn == nil {
		return fmt.Errorf("Subscribe() failed; JetstreamTriggerConn is nil")
	}

	var err error
	log := conn.Logger
	// derive subjects that we'll subscribe with using the dependencies passed in
	subjects := make(map[string]eventbuscommon.Dependency)
	for _, dep := range conn.deps {
		subjects[fmt.Sprintf("default.%s.%s", dep.EventSourceName, dep.EventName)] = dep
	}

	if !lastResetTime.IsZero() {
		err = conn.clearAllDependencies(&lastResetTime)
		if err != nil {
			errStr := fmt.Sprintf("failed to clear all dependencies as a result of condition reset time; err=%v", err)
			log.Error(errStr)
		}
	}

	ch := make(chan *nats.Msg) // channel with no buffer (I believe this should be okay - we will block writing messages to this channel while a message is still being processed but volume of messages shouldn't be so high as to cause a problem)
	wg := sync.WaitGroup{}
	processMsgsCloseCh := make(chan struct{})
	pullSubscribeCloseCh := make(map[string]chan struct{}, len(subjects))

	subscriptions := make([]*nats.Subscription, len(subjects))
	subscriptionIndex := 0

	// start the goroutines that will listen to the individual subscriptions
	for subject, dependency := range subjects {
		// set durable name separately for each subscription
		durableName := getDurableName(conn.sensorName, conn.triggerName, dependency.Name)

		conn.Logger.Debugf("durable name for sensor='%s', trigger='%s', dep='%s': '%s'", conn.sensorName, conn.triggerName, dependency.Name, durableName)
		log.Infof("Subscribing to subject %s with durable name %s", subject, durableName)
		subscriptions[subscriptionIndex], err = conn.JSContext.PullSubscribe(subject, durableName, nats.AckExplicit(), nats.DeliverNew())
		if err != nil {
			errorStr := fmt.Sprintf("Failed to subscribe to subject %s using group %s: %v", subject, durableName, err)
			log.Error(errorStr)
			return fmt.Errorf(errorStr)
		} else {
			log.Debugf("successfully subscribed to subject %s with durable name %s", subject, durableName)
		}

		pullSubscribeCloseCh[subject] = make(chan struct{})
		go conn.pullSubscribe(subscriptions[subscriptionIndex], ch, pullSubscribeCloseCh[subject], &wg)
		wg.Add(1)
		log.Debug("adding 1 to WaitGroup (pullSubscribe)")

		subscriptionIndex++
	}

	// create a single goroutine which which handle receiving messages to ensure that all of the processing is occurring on that
	// one goroutine and we don't need to worry about race conditions
	go conn.processMsgs(ch, processMsgsCloseCh, resetConditionsCh, transform, filter, action, &wg)
	wg.Add(1)
	log.Debug("adding 1 to WaitGroup (processMsgs)")

	for {
		select {
		case <-ctx.Done():
			log.Info("exiting, closing connection...")
			conn.shutdownSubscriptions(processMsgsCloseCh, pullSubscribeCloseCh, &wg)
			return nil
		case <-closeCh:
			log.Info("closing connection...")
			conn.shutdownSubscriptions(processMsgsCloseCh, pullSubscribeCloseCh, &wg)
			return nil
		}
	}
}

func (conn *JetstreamTriggerConn) shutdownSubscriptions(processMsgsCloseCh chan struct{}, pullSubscribeCloseCh map[string]chan struct{}, wg *sync.WaitGroup) {
	processMsgsCloseCh <- struct{}{}
	for _, ch := range pullSubscribeCloseCh {
		ch <- struct{}{}
	}
	wg.Wait()
	conn.NATSConn.Close()
	conn.Logger.Debug("closed NATSConn")
}

func (conn *JetstreamTriggerConn) pullSubscribe(
	subscription *nats.Subscription,
	msgChannel chan<- *nats.Msg,
	closeCh <-chan struct{},
	wg *sync.WaitGroup) {
	var previousErr error
	var previousErrTime time.Time

	for {
		// call Fetch with timeout
		msgs, fetchErr := subscription.Fetch(1, nats.MaxWait(time.Second*1))
		if fetchErr != nil && !errors.Is(fetchErr, nats.ErrTimeout) {
			if previousErr != fetchErr || time.Since(previousErrTime) > 10*time.Second {
				// avoid log spew - only log error every 10 seconds
				conn.Logger.Errorf("failed to fetch messages for subscription %+v, %v, previousErr=%v, previousErrTime=%v", subscription, fetchErr, previousErr, previousErrTime)
			}
			previousErr = fetchErr
			previousErrTime = time.Now()
		}

		// read from close channel but don't block if it's empty
		select {
		case <-closeCh:
			wg.Done()
			conn.Logger.Debug("wg.Done(): pullSubscribe")
			conn.Logger.Infof("exiting pullSubscribe() for subscription %+v", subscription)
			return
		default:
		}
		if fetchErr != nil && !errors.Is(fetchErr, nats.ErrTimeout) {
			continue
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
	wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		conn.Logger.Debug("wg.Done(): processMsgs")
	}()

	for {
		select {
		case msg := <-receiveChannel:
			conn.processMsg(msg, transform, filter, action)
		case <-resetConditionsCh:
			conn.Logger.Info("reset conditions")
			_ = conn.clearAllDependencies(nil)
		case <-closeCh:
			conn.Logger.Info("shutting down processMsgs routine")
			return
		}
	}
}

func (conn *JetstreamTriggerConn) processMsg(
	m *nats.Msg,
	transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
	filter func(string, cloudevents.Event) bool,
	action func(map[string]cloudevents.Event)) {
	meta, err := m.Metadata()
	if err != nil {
		conn.Logger.Errorf("can't get Metadata() for message %+v??", m)
	}

	ch := make(chan bool)
	go func () {
	  ticker := time.NewTicker(500 * time.Millisecond)
	  for {
	    select {
	    case <- ch:
	      err = m.AckSync()
	      ticker.Stop()
	      break
	    case <- ticker.C:
	      err = m.InProgress()
	    }
	  }
	}()

	defer func() {
	   ch <- true
	 }()
	 
	log := conn.Logger

	var event *cloudevents.Event
	if err := json.Unmarshal(m.Data, &event); err != nil {
		log.Errorf("Failed to convert to a cloudevent, discarding it... err: %v", err)
		return
	}

	// De-duplication
	// In the off chance that we receive the same message twice, don't re-process
	_, alreadyReceived := conn.recentMsgsByID[event.ID()]
	if alreadyReceived {
		log.Debugf("already received message of ID %d, ignore this", event.ID())
		return
	}

	// get all dependencies for this Trigger that match
	depNames, err := conn.getDependencyNames(event.Source(), event.Subject())
	if err != nil || len(depNames) == 0 {
		log.Errorf("Failed to get the dependency names, discarding it... err: %v", err)
		return
	}

	log.Debugf("New incoming Event Source Message, dependency names=%s, Stream seq: %s:%d, Consumer seq: %s:%d",
		depNames, meta.Stream, meta.Sequence.Stream, meta.Consumer, meta.Sequence.Consumer)

	for _, depName := range depNames {
		conn.processDependency(m, event, depName, transform, filter, action)
	}

	// Save message for de-duplication purposes
	conn.storeMessageID(event.ID())
	conn.purgeOldMsgs()
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
		for prevDep := range prevMsgs {
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
			log.Debugf("dependency expression successfully evaluated to true: '%s'", conn.dependencyExpression)

			messages := make(map[string]cloudevents.Event, len(prevMsgs)+1)
			for prevDep, msgInfo := range prevMsgs {
				messages[prevDep] = *msgInfo.Event
			}
			messages[depName] = *event
			log.Infof("Triggering actions after receiving dependency %s", depName)

			action(messages)

			_ = conn.clearAllDependencies(nil)
		} else {
			log.Debugf("dependency expression false: %s", conn.dependencyExpression)
			msgMetadata, err := m.Metadata()
			if err != nil {
				errStr := fmt.Sprintf("message %+v is not a jetstream message???: %v", m, err)
				log.Error(errStr)
				return
			}
			_ = conn.saveDependency(depName,
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
				return MsgInfo{}, true, fmt.Errorf(errStr)
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
		return fmt.Errorf(errorStr)
	}
	key := getDependencyKey(conn.triggerName, depName)

	_, err = conn.keyValueStore.Put(key, jsonEncodedMsg)
	if err != nil {
		errorStr := fmt.Sprintf("failed to store dependency under key %s, value:%s: %+v", key, jsonEncodedMsg, err)
		log.Error(errorStr)
		return fmt.Errorf(errorStr)
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
		return nil, fmt.Errorf(errStr)
	}

	return deps, nil
}

// save the message in our recent messages list (for de-duplication purposes)
func (conn *JetstreamTriggerConn) storeMessageID(id string) {
	now := time.Now().UnixNano()
	saveMsg := &msg{msgID: id, time: now}
	conn.recentMsgsByID[id] = saveMsg
	conn.recentMsgsByTime = append(conn.recentMsgsByTime, saveMsg)
}

func (conn *JetstreamTriggerConn) purgeOldMsgs() {
	now := time.Now().UnixNano()

	// evict any old messages from our message cache
	for _, msg := range conn.recentMsgsByTime {
		if now-msg.time > 60*1000*1000*1000 { // older than 1 minute
			conn.Logger.Debugf("deleting message %v from cache", *msg)
			delete(conn.recentMsgsByID, msg.msgID)
			conn.recentMsgsByTime = conn.recentMsgsByTime[1:]
		} else {
			break // these are ordered by time so we can break when we hit one that's still valid
		}
	}
}
