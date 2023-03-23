package base

import "go.uber.org/zap"

type KafkaConnection struct {
	Logger *zap.SugaredLogger
}

func NewKafkaConnection(logger *zap.SugaredLogger) *KafkaConnection {
	return &KafkaConnection{
		Logger: logger,
	}
}
