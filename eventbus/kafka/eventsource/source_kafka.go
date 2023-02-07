package eventsource

import (
	"strings"

	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/eventbus/common"
	"github.com/argoproj/argo-events/eventbus/kafka/base"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"go.uber.org/zap"
)

type KafkaSource struct {
	*base.Kafka
	topic string
}

func NewKafkaSource(kafkaConfig *eventbusv1alpha1.KafkaConfig, logger *zap.SugaredLogger) *KafkaSource {
	return &KafkaSource{
		base.NewKafka(strings.Split(kafkaConfig.URL, ","), logger),
		kafkaConfig.Topic,
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
