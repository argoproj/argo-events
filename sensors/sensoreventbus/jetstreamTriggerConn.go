package sensoreventbus

import (
	"context"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"

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
	action func(map[string]cloudevents.Event)) error {
	log := conn.Logger //.With("clientID", stream.clientID)

	group := ?

	// Create a Consumer
	_, err := conn.JSContext.AddConsumer("default", &nats.ConsumerConfig{
		Durable: group,
	})
	if err != nil {
		// tbd
	}

	// derive subjects that we'll subscribe with using the dependencies passed in
	subjects := make(map[string]struct{}) // essentially a set
	for _, dep := range dependencies {
		subjects[fmt.Sprintf("default.%s.%s", dep.EventSourceName, dep.EventName)] = struct{}{}
	}

	conn.SetupKeyValueStore()

	err = conn.CleanUpOnStart(group, sensorName, triggerName, dependencies)

	return nil
}

func (conn *JetstreamTriggerConn) CleanUpOnStart(group string, dependencies []Dependency) error {
	// first look in K/V store for old Triggers that no longer exist

	// for each

}
