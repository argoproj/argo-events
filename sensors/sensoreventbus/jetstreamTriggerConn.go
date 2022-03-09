package sensoreventbus

import (
	"context"
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

func (conn *JetstreamTriggerConn) Subscribe(ctx context.Context,
	closeCh <-chan struct{},
	resetConditionsCh <-chan struct{},
	lastResetTime time.Time,
	transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
	filter func(string, cloudevents.Event) bool,
	action func(map[string]cloudevents.Event),
	defaultSubject *string) error {
	/*log := conn.Logger.With("clientID", conn.ClientID())

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

	conn.SetupKeyValueStore()

	err = conn.CleanUpOnStart(group)*/

	return nil
}

func (n *JetstreamTriggerConn) getGroupNameFromClientID(clientID string) (string, error) {
	log := n.Logger.With("clientID", n.ClientID())
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
	// first look in K/V store for old Triggers that no longer exist

	// for each

	return nil
}
