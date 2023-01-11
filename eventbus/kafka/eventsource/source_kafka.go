package eventsource

import (
	"github.com/argoproj/argo-events/eventbus/common"
	"github.com/argoproj/argo-events/eventbus/kafka/base"
	"go.uber.org/zap"
)

type KafkaSource struct {
	*base.Kafka
}

func NewKafkaSource(brokers []string, logger *zap.SugaredLogger) *KafkaSource {
	return &KafkaSource{
		base.NewKafka(brokers, logger),
	}
}

func (s *KafkaSource) Initialize() error {
	return nil
}

func (s *KafkaSource) Connect(string) (common.EventSourceConnection, error) {
	conn, err := s.MakeConnection()
	if err != nil {
		return nil, err
	}

	return &KafkaSourceConnection{conn}, nil
}
