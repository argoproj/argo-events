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
	topic    string
	client   sarama.Client
	producer sarama.SyncProducer
}

func (c *KafkaSourceConnection) Publish(ctx context.Context, msg common.Message) error {
	key := msg.EventSourceName + ":" + msg.EventName

	partition, offset, err := c.producer.SendMessage(&sarama.ProducerMessage{
		Topic: c.topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(msg.Body),
	})

	if err != nil {
		return err
	}

	c.Logger.Infow("Published message to kafka", zap.String("topic", c.topic), zap.Int32("partition", partition), zap.Int64("offset", offset))

	return nil
}

func (c *KafkaSourceConnection) Close() error {
	if err := c.producer.Close(); err != nil {
		return err
	}

	return c.client.Close()
}

func (c *KafkaSourceConnection) IsClosed() bool {
	return c.client.Closed()
}
