package kafka

import (
	"sync"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"
)

type KafkaHandler struct {
	*sync.Mutex
	Logger    *zap.SugaredLogger
	GroupName string
	Producer  sarama.AsyncProducer
	Handlers  map[string]func(*sarama.ConsumerMessage) *KafkaTransaction
}

func (h *KafkaHandler) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *KafkaHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *KafkaHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg := <-claim.Messages():
			h.Logger.Infow("Received message",
				zap.String("topic", msg.Topic),
				zap.Int32("partition", msg.Partition),
				zap.Int64("offset", msg.Offset))

			// todo: fix
			if msg.Offset == 0 && claim.InitialOffset() == -1 {
				h.initOffset(session, msg.Topic, msg.Partition)
			}

			if handler, ok := h.Handlers[msg.Topic]; ok {
				transaction := handler(msg)
				if err := h.commit(transaction, msg, session); err != nil {
					h.Logger.Errorw("Transaction error", zap.Error(err))
				}
			}
		case <-session.Context().Done():
			return nil
		}
	}
}

func (h *KafkaHandler) Close() error {
	h.Lock()
	defer h.Unlock()

	return h.Producer.Close()
}

func (h *KafkaHandler) commit(transaction *KafkaTransaction, msg *sarama.ConsumerMessage, session sarama.ConsumerGroupSession) error {
	h.Lock()
	defer h.Unlock()

	return transaction.Commit(h.Producer, h.GroupName, msg, session, h.Logger)
}

func (h *KafkaHandler) initOffset(session sarama.ConsumerGroupSession, topic string, partition int32) {
	h.Lock()
	defer h.Unlock()

	session.MarkOffset(topic, partition, 0, "")
	session.Commit()
}
