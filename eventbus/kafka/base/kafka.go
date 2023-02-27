package base

import (
	"strings"

	"github.com/Shopify/sarama"
	"github.com/argoproj/argo-events/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"go.uber.org/zap"
)

type Kafka struct {
	Logger *zap.SugaredLogger
	config *eventbusv1alpha1.KafkaConfig
}

func NewKafka(config *eventbusv1alpha1.KafkaConfig, logger *zap.SugaredLogger) *Kafka {
	// set defaults
	if config.ConsumerGroup == nil {
		config.ConsumerGroup = &eventbusv1alpha1.KafkaConsumerGroup{}
	}

	return &Kafka{
		Logger: logger,
		config: config,
	}
}

func (k *Kafka) Brokers() []string {
	return strings.Split(k.config.URL, ",")
}

func (k *Kafka) Config() (*sarama.Config, error) {
	config := sarama.NewConfig()

	// consumer config
	config.Consumer.IsolationLevel = sarama.ReadCommitted
	config.Consumer.Offsets.AutoCommit.Enable = false

	switch k.config.ConsumerGroup.StartOldest {
	case true:
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	case false:
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	switch k.config.ConsumerGroup.RebalanceStrategy {
	case "sticky":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategySticky}
	case "roundrobin":
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
	default:
		config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
	}

	// producer config
	config.Producer.Idempotent = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Net.MaxOpenRequests = 1

	// common config
	if k.config.Version != "" {
		version, err := sarama.ParseKafkaVersion(k.config.Version)
		if err != nil {
			return nil, err
		}

		config.Version = version
	}

	// sasl
	if k.config.SASL != nil {
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLMechanism(k.config.SASL.GetMechanism())

		switch config.Net.SASL.Mechanism {
		case "SCRAM-SHA-512":
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &common.XDGSCRAMClient{HashGeneratorFcn: common.SHA512New}
			}
		case "SCRAM-SHA-256":
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &common.XDGSCRAMClient{HashGeneratorFcn: common.SHA256New}
			}
		}

		user, err := common.GetSecretFromVolume(k.config.SASL.UserSecret)
		if err != nil {
			return nil, err
		}
		config.Net.SASL.User = user

		password, err := common.GetSecretFromVolume(k.config.SASL.PasswordSecret)
		if err != nil {
			return nil, err
		}
		config.Net.SASL.Password = password
	}

	// tls
	if k.config.TLS != nil {
		tls, err := common.GetTLSConfig(k.config.TLS)
		if err != nil {
			return nil, err
		}

		config.Net.TLS.Config = tls
		config.Net.TLS.Enable = true
	}

	return config, nil
}
