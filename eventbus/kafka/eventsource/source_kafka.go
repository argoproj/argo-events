package eventsource

import (
	"fmt"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/common"
	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	"github.com/argoproj/argo-events/eventbus/kafka/base"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"go.uber.org/zap"
)

type KafkaSource struct {
	*base.Kafka
	config *sarama.Config
	topic  string
}

func NewKafkaSource(kafkaConfig *eventbusv1alpha1.KafkaConfig, logger *zap.SugaredLogger) *KafkaSource {
	config := sarama.NewConfig()
	config.Producer.Return.Errors = true
	config.Producer.Return.Successes = true
	// config.Net.TLS.Enable = kafkaConfig.TLS != nil

	var consumerGroup *eventbusv1alpha1.KafkaConsumerGroup
	switch kafkaConfig.ConsumerGroup {
	case nil:
		consumerGroup = &eventbusv1alpha1.KafkaConsumerGroup{}
	default:
		consumerGroup = kafkaConfig.ConsumerGroup
	}

	switch consumerGroup.RebalanceStrategy {
	case "sticky":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategySticky}
	case "roundrobin":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
	default:
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	}

	if kafkaConfig.SASL != nil {
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLMechanism(kafkaConfig.SASL.GetMechanism())
		if config.Net.SASL.Mechanism == "SCRAM-SHA-512" {
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &common.XDGSCRAMClient{HashGeneratorFcn: common.SHA512New} }
		} else if config.Net.SASL.Mechanism == "SCRAM-SHA-256" {
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &common.XDGSCRAMClient{HashGeneratorFcn: common.SHA256New} }
		}

		user, err := common.GetSecretFromVolume(kafkaConfig.SASL.UserSecret)
		if err != nil {
			fmt.Printf("error getting user value from secret, %v", err)
			return nil
		}
		config.Net.SASL.User = user

		password, err := common.GetSecretFromVolume(kafkaConfig.SASL.PasswordSecret)
		if err != nil {
			fmt.Printf("error getting password value from secret, %v", err)
			return nil
		}
		config.Net.SASL.Password = password
		config.Net.TLS.Enable = true
	}

	return &KafkaSource{
		base.NewKafka(strings.Split(kafkaConfig.URL, ","), logger),
		config,
		kafkaConfig.Topic,
	}
}

func (s *KafkaSource) Initialize() error {
	return nil
}

func (s *KafkaSource) Connect(string) (eventbuscommon.EventSourceConnection, error) {
	client, err := sarama.NewClient(s.Brokers, s.config)
	if err != nil {
		fmt.Println(err)
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
