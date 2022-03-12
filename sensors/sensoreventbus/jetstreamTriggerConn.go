package sensoreventbus

import (
	"context"
	"fmt"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"

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

	group, err := conn.getGroupNameFromClientID(conn.ClientID())
	if err != nil {
		return err
	}

	// Create a Consumer
	_, err = conn.JSContext.AddConsumer("default", &nats.ConsumerConfig{
		Durable: group,
	})
	if err != nil {
		// tbd
	}

	// derive subjects that we'll subscribe with using the dependencies passed in
	subjects := make(map[string]struct{}) // essentially a set
	for _, dep := range conn.deps {
		subjects[fmt.Sprintf("default.%s.%s", dep.EventSourceName, dep.EventName)] = struct{}{}
	}

	err = conn.CleanUpOnStart(group)

	// create a goroutine which which handle receiving messages to ensure that all of the processing is occurring on that
	// one goroutine and we don't need to worry about race conditions
	ch := make(chan *nats.Msg, 64) // todo: 64 is random - make a constant? any concerns about it not being big enough?
	for subject, _ := range subjects {
		_, err = conn.JSContext.ChanQueueSubscribe(subject, group, ch, nats.AckAll()) // todo: what other subscription options here?; also, do we need the Subscription returned by this call?
		if err != nil {
			log.Errorf("Failed to subscribe to subject %s using group %s: %v", subject, group, err)
		}
	}
	go conn.processMsgs(ctx, ch, closeCh)

	return nil
}

func (conn *JetstreamTriggerConn) processMsgs(ctx context.Context, receiveChannel <-chan *nats.Msg, closeCh <-chan struct{}) {

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
	conn.Logger.Errorf("received new message: %+v", msg)
}

func (conn *JetstreamTriggerConn) getGroupNameFromClientID(clientID string) (string, error) {
	log := conn.Logger.With("clientID", conn.ClientID())
	// take off the last part: clientID should have a dash at the end and we can remove that part
	strs := strings.Split(clientID, "-")
	if len(strs) < 2 {
		err := errors.Errorf("Expected client ID to contain dash: %s", clientID)
		log.Error(err)
		return "", err
	}
	return strings.Join(strs[:len(strs)-1], "-"), nil
}

func (conn *JetstreamTriggerConn) CleanUpOnStart(group string) error {
	// look in K/V store for Trigger expressions that have changed

	// for each Trigger that no longer exists, need to handle:
	// - messages sent for that Trigger that are in the K/V store
	// - messages sent to that Trigger that never reached it and are waiting in the eventbus (need to make a new connection and Drain())

	return nil
}
