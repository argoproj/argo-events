package base

import "go.uber.org/zap"

type Kafka struct {
	brokers []string
	logger  *zap.SugaredLogger
}

func NewKafka(brokers []string, logger *zap.SugaredLogger) *Kafka {
	return &Kafka{
		brokers: brokers,
		logger:  logger,
	}
}

func (k *Kafka) MakeConnection() (*KafkaConnection, error) {
	conn := &KafkaConnection{
		logger: k.logger,
	}

	return conn, nil
}
