package eventsource

import (
	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/eventbus/common"
	"github.com/argoproj/argo-events/eventbus/kafka/base"
	"go.uber.org/zap"
)

type KafkaSource struct {
	*base.Kafka
	topic string
}

func NewKafkaSource(brokers []string, topic string, logger *zap.SugaredLogger) *KafkaSource {
	return &KafkaSource{
		base.NewKafka(brokers, logger),
		topic,
	}
}

func (s *KafkaSource) Initialize() error {
	return nil
}

func (s *KafkaSource) Connect(string) (common.EventSourceConnection, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Errors = true
	config.Producer.Return.Successes = true

	client, err := sarama.NewClient(s.Brokers, config)
	if err != nil {
		return nil, err
	}

	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, err
	}

	conn := &KafkaSourceConnection{
		base.NewKafkaConnection(s.Logger),
		s.topic,
		client,
		producer,
	}

	return conn, nil
}
