package kafka

import (
	"time"

	"github.com/Knetic/govaluate"
	"github.com/argoproj/argo-events/eventbus/common"
	"github.com/argoproj/argo-events/eventbus/kafka/base"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"
)

type KafkaTriggerHandler interface {
	common.TriggerConnection
	Name() string
	Ready() bool
	Reset()
	OneAndDone() bool
	DependsOn(*cloudevents.Event) (string, bool)
	Transform(string, *cloudevents.Event) (*cloudevents.Event, error)
	Filter(string, *cloudevents.Event) bool
	Update(event *cloudevents.Event, partition int32, offset int64, timestamp time.Time) ([]*cloudevents.Event, error)
	Offset(int32, int64) int64
	Action([]*cloudevents.Event) func()
}

func (c *KafkaTriggerConnection) Name() string {
	return c.triggerName
}

func (c *KafkaTriggerConnection) Ready() bool {
	// cannot process events until the subscribe function has been
	// called, which is when these functions are set
	return c.transform != nil && c.filter != nil && c.action != nil
}

func (c *KafkaTriggerConnection) DependsOn(event *cloudevents.Event) (string, bool) {
	if dep, ok := c.dependencies[base.EventKey(event.Source(), event.Subject())]; ok {
		return dep.Name, true
	}

	return "", false
}

func (c *KafkaTriggerConnection) OneAndDone() bool {
	for _, token := range c.depExpression.Tokens() {
		if token.Kind == govaluate.LOGICALOP && token.Value == "&&" {
			return false
		}
	}

	return true
}

func (c *KafkaTriggerConnection) Transform(depName string, event *cloudevents.Event) (*cloudevents.Event, error) {
	return c.transform(depName, *event)
}

func (c *KafkaTriggerConnection) Filter(depName string, event *cloudevents.Event) bool {
	return c.filter(depName, *event)
}

func (c *KafkaTriggerConnection) Update(event *cloudevents.Event, partition int32, offset int64, timestamp time.Time) ([]*cloudevents.Event, error) {
	eventWithMetadata := &eventWithMetadata{
		Event:     event,
		partition: partition,
		offset:    offset,
		timestamp: timestamp,
	}

	// remove previous events with same source and subject and remove
	// all events older than last condition reset time
	i := 0
	for _, event := range c.events {
		if !event.Same(eventWithMetadata) && event.After(c.lastResetTime) {
			c.events[i] = event
			i++
		}
	}
	for j := i; j < len(c.events); j++ {
		c.events[j] = nil // avoid memory leak
	}
	c.events = append(c.events[:i], eventWithMetadata)

	satisfied, err := c.satisfied()
	if err != nil {
		return nil, err
	}

	// if satisfied, publish a message to the action topic containing
	// all events and reset the trigger
	var events []*cloudevents.Event
	if satisfied == true {
		defer c.Reset()
		for _, event := range c.events {
			events = append(events, event.Event)
		}
	}

	return events, nil
}

func (c *KafkaTriggerConnection) Offset(partition int32, offset int64) int64 {
	for _, event := range c.events {
		if partition == event.partition && offset > event.offset {
			offset = event.offset
		}
	}

	return offset
}

func (c *KafkaTriggerConnection) Action(events []*cloudevents.Event) func() {
	eventMap := map[string]cloudevents.Event{}
	for _, event := range events {
		if depName, ok := c.DependsOn(event); ok {
			eventMap[depName] = *event
		}
	}

	// If at least once is specified, we must call the action
	// function before committing a transaction, otherwise the
	// function must be called after. To call after we return a
	// function.
	var f func()
	if c.atLeastOnce {
		c.action(eventMap)
	} else {
		f = func() { c.action(eventMap) }
	}

	return f
}

func (c *KafkaTriggerConnection) satisfied() (interface{}, error) {
	parameters := Parameters{}
	for _, event := range c.events {
		if depName, ok := c.DependsOn(event.Event); ok {
			parameters[depName] = true
		}
	}

	c.Logger.Infow("Evaluating", zap.String("expr", c.depExpression.String()), zap.Any("parameters", parameters))

	return c.depExpression.Eval(parameters)
}

func (c *KafkaTriggerConnection) Reset() {
	c.events = nil
}

type Parameters map[string]bool

func (p Parameters) Get(name string) (interface{}, error) {
	return p[name], nil
}
