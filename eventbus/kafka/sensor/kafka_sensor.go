package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/Shopify/sarama"
	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	"github.com/argoproj/argo-events/eventbus/kafka/base"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"
)

type KafkaSensor struct {
	*base.Kafka
	*sync.Mutex
	sensor *sensorv1alpha1.Sensor

	// kafka details
	topics    *Topics
	client    sarama.Client
	consumer  sarama.ConsumerGroup
	hostname  string
	groupName string

	// triggers handlers
	// holds the state of all sensor triggers
	triggers Triggers

	// kafka handler
	// handles consuming from kafka, offsets, and transactions
	kafkaHandler *KafkaHandler
	connected    bool
}

func NewKafkaSensor(kafkaConfig *eventbusv1alpha1.KafkaBus, sensor *sensorv1alpha1.Sensor, hostname string, logger *zap.SugaredLogger) *KafkaSensor {
	topics := &Topics{
		event:   kafkaConfig.Topic,
		trigger: fmt.Sprintf("%s-%s-%s", kafkaConfig.Topic, sensor.Name, "trigger"),
		action:  fmt.Sprintf("%s-%s-%s", kafkaConfig.Topic, sensor.Name, "action"),
	}

	var groupName string
	if kafkaConfig.ConsumerGroup == nil || kafkaConfig.ConsumerGroup.GroupName == "" {
		groupName = fmt.Sprintf("%s-%s", sensor.Namespace, sensor.Name)
	} else {
		groupName = kafkaConfig.ConsumerGroup.GroupName
	}

	return &KafkaSensor{
		Kafka:     base.NewKafka(kafkaConfig, logger),
		Mutex:     &sync.Mutex{},
		sensor:    sensor,
		topics:    topics,
		hostname:  hostname,
		groupName: groupName,
		triggers:  Triggers{},
	}
}

type Topics struct {
	event   string
	trigger string
	action  string
}

func (t *Topics) List() []string {
	return []string{t.event, t.trigger, t.action}
}

type Triggers map[string]KafkaTriggerHandler

type TriggerWithDepName struct {
	KafkaTriggerHandler
	depName string
}

func (t Triggers) List(event *cloudevents.Event) []*TriggerWithDepName {
	triggers := []*TriggerWithDepName{}

	for _, trigger := range t {
		if depName, ok := trigger.DependsOn(event); ok {
			triggers = append(triggers, &TriggerWithDepName{trigger, depName})
		}
	}

	return triggers
}

func (t Triggers) Ready() bool {
	for _, trigger := range t {
		if !trigger.Ready() {
			return false
		}
	}
	return true
}

func (s *KafkaSensor) Initialize() error {
	config, err := s.Config()
	if err != nil {
		return err
	}

	// sensor specific config
	config.Producer.Transaction.ID = s.hostname

	client, err := sarama.NewClient(s.Brokers(), config)
	if err != nil {
		return err
	}

	consumer, err := sarama.NewConsumerGroupFromClient(s.groupName, client)
	if err != nil {
		return err
	}

	producer, err := sarama.NewAsyncProducerFromClient(client)
	if err != nil {
		return err
	}

	offsetManager, err := sarama.NewOffsetManagerFromClient(s.groupName, client)
	if err != nil {
		return err
	}

	s.client = client
	s.consumer = consumer
	s.kafkaHandler = &KafkaHandler{
		Mutex:         &sync.Mutex{},
		Logger:        s.Logger,
		GroupName:     s.groupName,
		Producer:      producer,
		OffsetManager: offsetManager,
		TriggerTopic:  s.topics.trigger,
		Handlers: map[string]func(*sarama.ConsumerMessage) ([]*sarama.ProducerMessage, int64, func()){
			s.topics.event:   s.Event,
			s.topics.trigger: s.Trigger,
			s.topics.action:  s.Action,
		},
	}

	return nil
}

func (s *KafkaSensor) Connect(ctx context.Context, triggerName string, depExpression string, dependencies []eventbuscommon.Dependency, atLeastOnce bool) (eventbuscommon.TriggerConnection, error) {
	s.Lock()
	defer s.Unlock()

	// connect only if disconnected, if ever the connection is lost
	// the connected boolean will flip and the sensor listener will
	// attempt to reconnect by invoking this function again
	if !s.connected {
		go s.Listen(ctx)
		s.connected = true
	}

	if _, ok := s.triggers[triggerName]; !ok {
		expr, err := govaluate.NewEvaluableExpression(strings.ReplaceAll(depExpression, "-", "\\-"))
		if err != nil {
			return nil, err
		}

		depMap := map[string]eventbuscommon.Dependency{}
		for _, dep := range dependencies {
			depMap[base.EventKey(dep.EventSourceName, dep.EventName)] = dep
		}

		s.triggers[triggerName] = &KafkaTriggerConnection{
			KafkaConnection: base.NewKafkaConnection(s.Logger),
			sensorName:      s.sensor.Name,
			triggerName:     triggerName,
			depExpression:   expr,
			dependencies:    depMap,
			atLeastOnce:     atLeastOnce,
			close:           s.Close,
			isClosed:        s.IsClosed,
		}
	}

	return s.triggers[triggerName], nil
}

func (s *KafkaSensor) Listen(ctx context.Context) {
	defer s.Disconnect()

	for {
		if len(s.triggers) != len(s.sensor.Spec.Triggers) || !s.triggers.Ready() {
			s.Logger.Info("Not ready to consume, waiting...")
			time.Sleep(3 * time.Second)
			continue
		}

		s.Logger.Infow("Consuming", zap.Strings("topics", s.topics.List()), zap.String("group", s.groupName))

		if err := s.consumer.Consume(ctx, s.topics.List(), s.kafkaHandler); err != nil {
			// fail fast if topics do not exist
			if err == sarama.ErrUnknownTopicOrPartition {
				s.Logger.Fatalf(
					"Topics do not exist. Please ensure the topics '%s' have been created, or the kafka setting '%s' is set to true.",
					s.topics.List(),
					"auto.create.topics.enable",
				)
			}

			s.Logger.Errorw("Failed to consume", zap.Error(err))
			return
		}

		if err := ctx.Err(); err != nil {
			s.Logger.Errorw("Kafka error", zap.Error(err))
			return
		}
	}
}

func (s *KafkaSensor) Disconnect() {
	s.Lock()
	defer s.Unlock()

	s.connected = false
}

func (s *KafkaSensor) Close() error {
	s.Lock()
	defer s.Unlock()

	// protect against being called multiple times
	if s.IsClosed() {
		return nil
	}

	if err := s.consumer.Close(); err != nil {
		return err
	}

	if err := s.kafkaHandler.Close(); err != nil {
		return err
	}

	return s.client.Close()
}

func (s *KafkaSensor) IsClosed() bool {
	return !s.connected || s.client.Closed()
}

func (s *KafkaSensor) Event(msg *sarama.ConsumerMessage) ([]*sarama.ProducerMessage, int64, func()) {
	var event *cloudevents.Event
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		s.Logger.Errorw("Failed to deserialize cloudevent, skipping", zap.Error(err))
		return nil, msg.Offset + 1, nil
	}

	messages := []*sarama.ProducerMessage{}
	for _, trigger := range s.triggers.List(event) {
		event, err := trigger.Transform(trigger.depName, event)
		if err != nil {
			s.Logger.Errorw("Failed to transform cloudevent, skipping", zap.Error(err))
			continue
		}

		if !trigger.Filter(trigger.depName, event) {
			s.Logger.Debug("Filter condition satisfied, skipping")
			continue
		}

		// if the trigger only requires one message to be invoked we
		// can skip ahead to the action topic, otherwise produce to
		// the trigger topic

		var data any
		var topic string
		if trigger.OneAndDone() {
			data = []*cloudevents.Event{event}
			topic = s.topics.action
		} else {
			data = event
			topic = s.topics.trigger
		}

		value, err := json.Marshal(data)
		if err != nil {
			s.Logger.Errorw("Failed to serialize cloudevent, skipping", zap.Error(err))
			continue
		}

		messages = append(messages, &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(trigger.Name()),
			Value: sarama.ByteEncoder(value),
		})
	}

	return messages, msg.Offset + 1, nil
}

func (s *KafkaSensor) Trigger(msg *sarama.ConsumerMessage) ([]*sarama.ProducerMessage, int64, func()) {
	var event *cloudevents.Event
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		// do not return here as we still need to call trigger.Offset
		// below to determine current offset
		s.Logger.Errorw("Failed to deserialize cloudevent, skipping", zap.Error(err))
	}

	messages := []*sarama.ProducerMessage{}
	offset := msg.Offset + 1

	// update trigger with new event and add any resulting action to
	// transaction messages
	if trigger, ok := s.triggers[string(msg.Key)]; ok && event != nil {
		func() {
			events, err := trigger.Update(event, msg.Partition, msg.Offset, msg.Timestamp)
			if err != nil {
				s.Logger.Errorw("Failed to update trigger, skipping", zap.Error(err))
				return
			}

			// no events, trigger not yet satisfied
			if events == nil {
				return
			}

			value, err := json.Marshal(events)
			if err != nil {
				s.Logger.Errorw("Failed to serialize cloudevent, skipping", zap.Error(err))
				return
			}

			messages = append(messages, &sarama.ProducerMessage{
				Topic: s.topics.action,
				Key:   sarama.StringEncoder(trigger.Name()),
				Value: sarama.ByteEncoder(value),
			})
		}()
	}

	// need to determine smallest possible offset against all
	// triggers as other triggers may have messages that land on the
	// same partition
	for _, trigger := range s.triggers {
		offset = trigger.Offset(msg.Partition, offset)
	}

	return messages, offset, nil
}

func (s *KafkaSensor) Action(msg *sarama.ConsumerMessage) ([]*sarama.ProducerMessage, int64, func()) {
	var events []*cloudevents.Event
	if err := json.Unmarshal(msg.Value, &events); err != nil {
		s.Logger.Errorw("Failed to deserialize cloudevents, skipping", zap.Error(err))
		return nil, msg.Offset + 1, nil
	}

	var f func()
	if trigger, ok := s.triggers[string(msg.Key)]; ok {
		f = trigger.Action(events)
	}

	return nil, msg.Offset + 1, f
}
