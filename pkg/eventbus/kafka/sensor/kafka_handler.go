package kafka

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/eventbus/kafka/base"
)

type KafkaHandler struct {
	*sync.Mutex
	Logger *zap.SugaredLogger

	// kafka details
	GroupName     string
	Producer      sarama.AsyncProducer
	OffsetManager sarama.OffsetManager
	TriggerTopic  string

	// BatchMaxWait is the maximum time to wait for a batch of messages.
	// When set to 0, messages are processed individually in real-time.
	BatchMaxWait time.Duration

	// handler functions
	// one function for each consumed topic, return messages, an
	// offset and an optional function that will in a transaction
	Handlers map[string]func(*sarama.ConsumerMessage) ([]*sarama.ProducerMessage, int64, func())

	// cleanup function
	// used to clear state when consumer group is rebalanced
	Reset func() error

	// maintains a mapping of keys (which correspond to triggers)
	// to offsets, used to ensure triggers aren't invoked twice
	checkpoints Checkpoints
}

type Checkpoints map[string]map[int32]*Checkpoint

type Checkpoint struct {
	Logger  *zap.SugaredLogger
	Init    bool
	Offsets map[string]int64
}

func (c *Checkpoint) Skip(key string, offset int64) bool {
	if c.Offsets == nil {
		return false
	}
	return offset < c.Offsets[key]
}

func (c *Checkpoint) Set(key string, offset int64) {
	if c.Offsets == nil {
		c.Offsets = map[string]int64{}
	}
	c.Offsets[key] = offset
}

func (c *Checkpoint) Metadata() string {
	if c.Offsets == nil {
		return ""
	}

	metadata, err := json.Marshal(c.Offsets)
	if err != nil {
		c.Logger.Errorw("Failed to serialize metadata", err)
		return ""
	}

	return string(metadata)
}

func (h *KafkaHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.Logger.Infow("Kafka setup", zap.Any("claims", session.Claims()))

	// instantiates checkpoints for all topic/partitions managed by
	// this claim
	h.checkpoints = Checkpoints{}

	for topic, partitions := range session.Claims() {
		h.checkpoints[topic] = map[int32]*Checkpoint{}

		for _, partition := range partitions {
			partitionOffsetManager, err := h.OffsetManager.ManagePartition(topic, partition)
			if err != nil {
				return err
			}

			func() {
				var offsets map[string]int64

				defer partitionOffsetManager.AsyncClose()
				offset, metadata := partitionOffsetManager.NextOffset()

				// only need to manage the offsets for each trigger
				// with respect to the trigger topic
				if topic == h.TriggerTopic && metadata != "" {
					if err := json.Unmarshal([]byte(metadata), &offsets); err != nil {
						// if metadata is invalid json, it will be
						// reset to an empty map
						h.Logger.Errorw("Failed to deserialize metadata, resetting", err)
					}
				}

				h.checkpoints[topic][partition] = &Checkpoint{
					Logger:  h.Logger,
					Init:    offset == -1, // mark offset when first message consumed
					Offsets: offsets,
				}
			}()

			h.OffsetManager.Commit()
			if err := partitionOffsetManager.Close(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *KafkaHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.Logger.Infow("Kafka cleanup", zap.Any("claims", session.Claims()))
	return h.Reset()
}

func (h *KafkaHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	handler, ok := h.Handlers[claim.Topic()]
	if !ok {
		return fmt.Errorf("unrecognized topic %s", claim.Topic())
	}

	checkpoint, ok := h.checkpoints[claim.Topic()][claim.Partition()]
	if !ok {
		return fmt.Errorf("unrecognized topic %s or partition %d", claim.Topic(), claim.Partition())
	}

	if h.BatchMaxWait == 0 {
		return h.consumeClaimRealtime(session, claim, handler, checkpoint)
	}
	return h.consumeClaimBatched(session, claim, handler, checkpoint)
}

// consumeClaimRealtime processes messages one at a time without batching,
// providing the lowest possible latency.
func (h *KafkaHandler) consumeClaimRealtime(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
	handler func(*sarama.ConsumerMessage) ([]*sarama.ProducerMessage, int64, func()),
	checkpoint *Checkpoint,
) error {
	for {
		select {
		case msg := <-claim.Messages():
			if msg == nil {
				continue
			}

			key := string(msg.Key)

			h.Logger.Infow("Received message",
				zap.String("topic", msg.Topic),
				zap.String("key", key),
				zap.Int32("partition", msg.Partition),
				zap.Int64("offset", msg.Offset))

			if checkpoint.Init {
				session.MarkOffset(msg.Topic, msg.Partition, msg.Offset, "")
				session.Commit()
				checkpoint.Init = false
			}

			if checkpoint.Skip(key, msg.Offset) {
				h.Logger.Infof("Skipping trigger '%s' (%d<%d)", key, msg.Offset, checkpoint.Offsets[key])
				continue
			}

			messages, offset, f := handler(msg)
			if msg.Topic == h.TriggerTopic && len(messages) > 0 {
				checkpoint.Set(key, msg.Offset+1)
			}

			transaction := &KafkaTransaction{
				Logger:        h.Logger,
				Producer:      h.Producer,
				GroupName:     h.GroupName,
				Topic:         claim.Topic(),
				Partition:     claim.Partition(),
				ResetOffset:   msg.Offset,
				ResetMetadata: checkpoint.Metadata(),
			}

			func() {
				h.Lock()
				defer h.Unlock()
				if err := transaction.Commit(session, messages, offset, checkpoint.Metadata()); err != nil {
					h.Logger.Errorw("Transaction error", zap.Error(err))
				}
			}()

			if f != nil {
				go f()
			}
		case <-session.Context().Done():
			return nil
		}
	}
}

// consumeClaimBatched processes messages in batches, amortizing Kafka
// transaction overhead across multiple messages for better throughput.
func (h *KafkaHandler) consumeClaimBatched(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
	handler func(*sarama.ConsumerMessage) ([]*sarama.ProducerMessage, int64, func()),
	checkpoint *Checkpoint,
) error {
	batch := base.Batch(100, h.BatchMaxWait, claim.Messages())

	for {
		select {
		case msgs := <-batch:
			if len(msgs) == 0 {
				h.Logger.Warn("Kafka batch contains no messages")
				continue
			}

			transaction := &KafkaTransaction{
				Logger:        h.Logger,
				Producer:      h.Producer,
				GroupName:     h.GroupName,
				Topic:         claim.Topic(),
				Partition:     claim.Partition(),
				ResetOffset:   msgs[0].Offset,
				ResetMetadata: checkpoint.Metadata(),
			}

			var messages []*sarama.ProducerMessage
			var offset int64
			var fns []func()

			for _, msg := range msgs {
				key := string(msg.Key)

				h.Logger.Infow("Received message",
					zap.String("topic", msg.Topic),
					zap.String("key", key),
					zap.Int32("partition", msg.Partition),
					zap.Int64("offset", msg.Offset))

				if checkpoint.Init {
					session.MarkOffset(msg.Topic, msg.Partition, msg.Offset, "")
					session.Commit()
					checkpoint.Init = false
				}

				if checkpoint.Skip(key, msg.Offset) {
					h.Logger.Infof("Skipping trigger '%s' (%d<%d)", key, msg.Offset, checkpoint.Offsets[key])
					continue
				}

				m, o, f := handler(msg)
				if msg.Topic == h.TriggerTopic && len(m) > 0 {
					checkpoint.Set(key, msg.Offset+1)
				}

				messages = append(messages, m...)
				offset = o
				if f != nil {
					fns = append(fns, f)
				}
			}

			func() {
				h.Lock()
				defer h.Unlock()
				if err := transaction.Commit(session, messages, offset, checkpoint.Metadata()); err != nil {
					h.Logger.Errorw("Transaction error", zap.Error(err))
				}
			}()

			for _, fn := range fns {
				go fn()
			}
		case <-session.Context().Done():
			return nil
		}
	}
}

func (h *KafkaHandler) Close() error {
	h.Lock()
	defer h.Unlock()

	if err := h.OffsetManager.Close(); err != nil {
		return err
	}

	return h.Producer.Close()
}
