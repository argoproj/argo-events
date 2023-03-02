package eventsource

import (
	"context"

	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/eventbus/common"
	"github.com/argoproj/argo-events/eventbus/kafka/base"
	"go.uber.org/zap"
)

type KafkaSourceConnection struct {
	*base.KafkaConnection
	Topic    string
	Client   sarama.Client
	Producer sarama.SyncProducer
}

func (c *KafkaSourceConnection) Publish(ctx context.Context, msg common.Message) error {
	key := base.EventKey(msg.EventSourceName, msg.EventName)
	partition, offset, err := c.Producer.SendMessage(&sarama.ProducerMessage{
		Topic: c.Topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(msg.Body),
	})

	if err != nil {
		// fail fast if topic does not exist
		if err == sarama.ErrUnknownTopicOrPartition {
			c.Logger.Fatalf(
				"Topic does not exist. Please ensure the topic '%s' has been created, or the kafka setting '%s' is set to true.",
				c.Topic,
				"auto.create.topics.enable",
			)
		}

		return err
	}

	c.Logger.Infow("Published message to kafka", zap.String("topic", c.Topic), zap.String("key", key), zap.Int32("partition", partition), zap.Int64("offset", offset))

	return nil
}

func (c *KafkaSourceConnection) Close() error {
	if err := c.Producer.Close(); err != nil {
		return err
	}

	return c.Client.Close()
}

func (c *KafkaSourceConnection) IsClosed() bool {
	return c.Client.Closed()
}
