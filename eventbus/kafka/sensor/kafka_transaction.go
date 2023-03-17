package kafka

import (
	"github.com/Shopify/sarama"
	"go.uber.org/zap"
)

type KafkaTransaction struct {
	Logger *zap.SugaredLogger

	// kafka details
	Producer  sarama.AsyncProducer
	GroupName string
	Topic     string
	Partition int32

	// used to reset the offset and metadata if transaction fails
	ResetOffset   int64
	ResetMetadata string
}

func (t *KafkaTransaction) Commit(session sarama.ConsumerGroupSession, messages []*sarama.ProducerMessage, offset int64, metadata string) error {
	// No need for a transaction if no messages, just update the
	// offset and metadata
	if len(messages) == 0 {
		session.MarkOffset(t.Topic, t.Partition, offset, metadata)
		session.Commit()
		return nil
	}

	t.Logger.Infow("Begin transaction",
		zap.String("topic", t.Topic),
		zap.Int32("partition", t.Partition),
		zap.Int("messages", len(messages)))

	if err := t.Producer.BeginTxn(); err != nil {
		return err
	}

	for _, msg := range messages {
		t.Producer.Input() <- msg
	}

	offsets := map[string][]*sarama.PartitionOffsetMetadata{
		t.Topic: {{
			Partition: t.Partition,
			Offset:    offset,
			Metadata:  &metadata,
		}},
	}

	if err := t.Producer.AddOffsetsToTxn(offsets, t.GroupName); err != nil {
		t.Logger.Errorw("Kafka transaction error", zap.Error(err))
		t.handleTxnError(session, func() error {
			return t.Producer.AddOffsetsToTxn(offsets, t.GroupName)
		})
	}

	if err := t.Producer.CommitTxn(); err != nil {
		t.Logger.Errorw("Kafka transaction error", zap.Error(err))
		t.handleTxnError(session, func() error {
			return t.Producer.CommitTxn()
		})
	}

	t.Logger.Infow("Finished transaction",
		zap.String("topic", t.Topic),
		zap.Int32("partition", t.Partition))

	return nil
}

func (t *KafkaTransaction) handleTxnError(session sarama.ConsumerGroupSession, defaulthandler func() error) {
	for {
		if t.Producer.TxnStatus()&sarama.ProducerTxnFlagFatalError != 0 {
			// reset current consumer offset to retry consume this record
			session.ResetOffset(t.Topic, t.Partition, t.ResetOffset, t.ResetMetadata)
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
			session.ResetOffset(t.Topic, t.Partition, t.ResetOffset, t.ResetMetadata)
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
