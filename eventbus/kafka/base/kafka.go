package base

import "go.uber.org/zap"

type Kafka struct {
	Brokers []string
	Logger  *zap.SugaredLogger
}

func NewKafka(brokers []string, logger *zap.SugaredLogger) *Kafka {
	return &Kafka{
		Brokers: brokers,
		Logger:  logger,
	}
}
