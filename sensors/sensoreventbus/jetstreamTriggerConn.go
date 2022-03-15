package sensoreventbus

import (
	"context"
	"fmt"
	"sync"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"

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
	deps                 []Dependency
}

func NewJetstreamTriggerConn(conn *eventbusdriver.JetstreamConnection,
	sensorName string,
	triggerName string,
	keyValueStore *nats.KeyValue,
	dependencyExpression string,
	deps []Dependency) *JetstreamTriggerConn {
	connection := &JetstreamTriggerConn{conn, sensorName, triggerName, keyValueStore, dependencyExpression, deps}
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
	go conn.processMsgs(ctx, ch, processMsgsCloseCh, transform, filter, action, wg)
	wg.Add(1)

	for {
		select {
		case <-ctx.Done():
			log.Info("exiting, closing connection...")
			conn.NATSConn.Close() // todo: should this go here?
			processMsgsCloseCh <- struct{}{}
			wg.Wait()
			return nil
		case <-closeCh:
			log.Info("closing connection...")
			conn.NATSConn.Close()
			processMsgsCloseCh <- struct{}{}
			wg.Wait()
			return nil
		}
	}

	return nil
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
	ctx context.Context,
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
			conn.processMsg(msg)
		case <-ctx.Done():
			// todo
			return
		case <-closeCh:
			// todo
			return
		}
	}
}

func (conn *JetstreamTriggerConn) processMsg(msg *nats.Msg) {
	defer msg.Ack()
	conn.Logger.Infof("received new message: %+v", msg)
}

func (conn *JetstreamTriggerConn) CleanUpOnStart() error {
	// look in K/V store for Trigger expressions that have changed

	// for each Trigger that no longer exists, need to handle:
	// - messages sent for that Trigger that are in the K/V store
	// - messages sent to that Trigger that never reached it and are waiting in the eventbus (need to make a new connection and Drain())

	return nil
}
