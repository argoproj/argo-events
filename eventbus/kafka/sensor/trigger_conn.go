package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/argoproj/argo-events/eventbus/common"
	"github.com/argoproj/argo-events/eventbus/kafka/base"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type KafkaTriggerConnection struct {
	*base.KafkaConnection
	KafkaTriggerHandler

	sensorName    string
	triggerName   string
	depExpression *govaluate.EvaluableExpression
	dependencies  map[string]common.Dependency
	atLeastOnce   bool

	// functions
	close     func() error
	isClosed  func() bool
	transform func(string, cloudevents.Event) (*cloudevents.Event, error)
	filter    func(string, cloudevents.Event) bool
	action    func(map[string]cloudevents.Event)

	// state
	events []*eventWithPartitionAndOffset
}

type eventWithPartitionAndOffset struct {
	*cloudevents.Event
	partition int32
	offset    int64
}

func (c *KafkaTriggerConnection) String() string {
	return fmt.Sprintf("KafkaTriggerConnection{Sensor:%s,Trigger:%s}", c.sensorName, c.triggerName)
}

func (c *KafkaTriggerConnection) Close() error {
	return c.close()
}

func (c *KafkaTriggerConnection) IsClosed() bool {
	return c.isClosed()
}

func (c *KafkaTriggerConnection) Subscribe(
	ctx context.Context,
	closeCh <-chan struct{},
	resetConditionsCh <-chan struct{},
	lastResetTime time.Time,
	transform func(depName string, event cloudevents.Event) (*cloudevents.Event, error),
	filter func(string, cloudevents.Event) bool,
	action func(map[string]cloudevents.Event),
	topic *string) error {
	c.transform = transform
	c.filter = filter
	c.action = action

	for {
		select {
		case <-ctx.Done():
			return c.Close()
		case <-closeCh:
			// this is a noop since a kafka connection is maintained
			// on the overall sensor vs indididual triggers
			return nil
		case <-resetConditionsCh:
			// todo: make resilient (bump offset)
			c.reset()
		}
	}
}
