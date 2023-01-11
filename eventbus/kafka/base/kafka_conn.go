package base

import "go.uber.org/zap"

type KafkaConnection struct {
	logger *zap.SugaredLogger
}

func (c *KafkaConnection) Close() error {
	return nil
}

func (c *KafkaConnection) IsClosed() bool {
	return false
}
