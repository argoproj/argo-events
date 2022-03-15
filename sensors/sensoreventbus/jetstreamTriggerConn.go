package sensoreventbus

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"

	"encoding/json"

	"github.com/argoproj/argo-events/common"
	eventbusdriver "github.com/argoproj/argo-events/eventbus/driver"
	nats "github.com/nats-io/nats.go"
)

type JetstreamTriggerConn struct {
	*eventbusdriver.JetstreamConnection
	sensorName           string
	triggerName          string
	keyValueStore        *nats.KeyValue
	dependencyExpression string
	requiresANDLogic     bool
	deps                 []Dependency
	sourceDepMap         map[string][]string // maps EventSource and EventName to dependency name
}

func NewJetstreamTriggerConn(conn *eventbusdriver.JetstreamConnection,
	sensorName string,
	triggerName string,
	keyValueStore *nats.KeyValue,
	dependencyExpression string,
	deps []Dependency) *JetstreamTriggerConn {

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
		keyValueStore:        keyValueStore,
		dependencyExpression: dependencyExpression,
		requiresANDLogic:     strings.Contains(dependencyExpression, "&"),
		deps:                 deps,
		sourceDepMap:         sourceDepMap}
	connection.Logger = connection.Logger.With("triggerName", connection.triggerName).With("clientID", connection.ClientID())
	return connection
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
	subjects := make(map[string]Dependency)
	for _, dep := range conn.deps {
		subjects[fmt.Sprintf("default.%s.%s", dep.EventSourceName, dep.EventName)] = dep
	}

	err := conn.CleanUpOnStart()
	if err != nil {
		return err
	}

	ch := make(chan *nats.Msg, 64) // todo: 64 is random - make a constant? any concerns about it not being big enough?
	wg := sync.WaitGroup{}
	processMsgsCloseCh := make(chan struct{})

	// create a goroutine which which handle receiving messages to ensure that all of the processing is occurring on that
	// one goroutine and we don't need to worry about race conditions
	subscriptions := make([]*nats.Subscription, len(subjects))
	subscriptionIndex := 0

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

	// this method will process the incoming messages
	go conn.processMsgs(ch, processMsgsCloseCh, transform, filter, action, wg)
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
	transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
	filter func(string, cloudevents.Event) bool,
	action func(map[string]cloudevents.Event),
	wg sync.WaitGroup) {

	defer wg.Done()

	for {
		select {
		case msg := <-receiveChannel:
			conn.processMsg(msg, transform, filter, action)
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

		parameters := make(map[string]bool, len(conn.deps))
		//messages := conn.getSavedDependencies()
		// derive parameters from messages

		parameters[depName] = true

		// if expression is true, trigger and clear the K/V store
		// else save the new message in the K/V store
	}
}

func (conn *JetstreamTriggerConn) getSavedDependencies() map[string]cloudevents.Event {
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

type msgInfo struct {
	seq       uint64
	timestamp int64
	event     *cloudevents.Event
	// timestamp of last delivered
	lastDeliveredTime int64
}
