package kafka

import (
	"github.com/Shopify/sarama"
	"go.uber.org/zap"
)

type KafkaTransaction struct {
	Logger    *zap.SugaredLogger
	Producer  sarama.AsyncProducer
	GroupName string

	Messages []*sarama.ProducerMessage
	Offset   int64
	Metadata string
	After    func() // will be invoked after the transaction is done
}

func (t *KafkaTransaction) Commit(msg *sarama.ConsumerMessage, session sarama.ConsumerGroupSession) error {
	// Defer the after function, used to implement at most once
	if t.After != nil {
		defer t.After()
	}

	// No need for a transaction if no messages
	if len(t.Messages) == 0 {
		session.MarkOffset(msg.Topic, msg.Partition, t.Offset, t.Metadata)
		session.Commit()
		return nil
	}

	t.Logger.Infow("Begin transaction",
		zap.String("topic", msg.Topic),
		zap.Int32("partition", msg.Partition),
		zap.Int("messages", len(t.Messages)))

	if err := t.Producer.BeginTxn(); err != nil {
		return err
	}

	for _, msg := range t.Messages {
		t.Producer.Input() <- msg
	}

	offsets := map[string][]*sarama.PartitionOffsetMetadata{
		msg.Topic: {{
			Partition: msg.Partition,
			Offset:    t.Offset,
			Metadata:  &t.Metadata,
		}},
	}

	if err := t.Producer.AddOffsetsToTxn(offsets, t.GroupName); err != nil {
		t.Logger.Errorw("Kafka transaction error", zap.Error(err))
		t.handleTxnError(msg, session, func() error {
			return t.Producer.AddOffsetsToTxn(offsets, t.GroupName)
		})
	}

	if err := t.Producer.CommitTxn(); err != nil {
		t.Logger.Errorw("Kafka transaction error", zap.Error(err))
		t.handleTxnError(msg, session, func() error {
			return t.Producer.CommitTxn()
		})
	}

	t.Logger.Infow("Finished transaction",
		zap.String("topic", msg.Topic),
		zap.Int32("partition", msg.Partition))

	return nil
}

func (t *KafkaTransaction) handleTxnError(msg *sarama.ConsumerMessage, session sarama.ConsumerGroupSession, defaulthandler func() error) {
	for {
		if t.Producer.TxnStatus()&sarama.ProducerTxnFlagFatalError != 0 {
			// reset current consumer offset to retry consume this record
			session.ResetOffset(msg.Topic, msg.Partition, msg.Offset, "")
			// fatal error, need to restart
			t.Logger.Fatal("Message consumer: t.Producer is in a fatal state.")
			return
		}
		if t.Producer.TxnStatus()&sarama.ProducerTxnFlagAbortableError != 0 {
			if err := t.Producer.AbortTxn(); err != nil {
				t.Logger.Errorw("Message consumer: unable to abort transaction.", zap.Error(err))
				continue
			}
			// reset current consumer offset to retry consume this record
			session.ResetOffset(msg.Topic, msg.Partition, msg.Offset, "")
			// fatal error, need to restart
			t.Logger.Fatal("Message consumer: t.Producer is in a fatal state, aborted transaction.")
			return
		}

		// attempt retry
		if err := defaulthandler(); err == nil {
			return
		}
	}
}
