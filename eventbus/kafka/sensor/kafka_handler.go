package kafka

import (
	"encoding/json"
	"sync"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"
)

type KafkaHandler struct {
	*sync.Mutex
	Logger        *zap.SugaredLogger
	GroupName     string
	Producer      sarama.AsyncProducer
	OffsetManager sarama.OffsetManager
	TriggerTopic  string
	Handlers      map[string]func(*sarama.ConsumerMessage) ([]*sarama.ProducerMessage, int64, func())
	checkpoints   map[string]map[int32]*Checkpoint
}

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
	metadata, err := json.Marshal(c.Offsets)
	if err != nil {
		c.Logger.Errorw("Failed to serialize metadata", err)
		return ""
	}

	return string(metadata)
}

func (h *KafkaHandler) Setup(session sarama.ConsumerGroupSession) error {
	// instantiate/reset
	h.checkpoints = map[string]map[int32]*Checkpoint{}

	for topic, partitions := range session.Claims() {
		h.checkpoints[topic] = map[int32]*Checkpoint{}

		for _, partition := range partitions {
			partitionOffsetManager, err := h.OffsetManager.ManagePartition(topic, partition)
			if err != nil {
				return err
			}

			func() {
				defer partitionOffsetManager.AsyncClose()
				offset, metadata := partitionOffsetManager.NextOffset()

				var offsets map[string]int64
				if metadata != "" {
					if err := json.Unmarshal([]byte(metadata), &offsets); err != nil {
						h.Logger.Errorw("Failed to deserialize metadata, resetting", err)
					}
				}

				h.checkpoints[topic][partition] = &Checkpoint{
					Logger:  h.Logger,
					Init:    offset == -1, // need to mark offset on first message
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

			if handler, ok := h.Handlers[msg.Topic]; ok {
				key := string(msg.Key)
				checkpoint := h.checkpoints[msg.Topic][msg.Partition]

				if checkpoint.Init {
					session.MarkOffset(msg.Topic, msg.Partition, msg.Offset, "")
					session.Commit()
					checkpoint.Init = false
				}

				if checkpoint.Skip(key, msg.Offset) {
					h.Logger.Infof("Skipping trigger '%s' (%d<%d)", key, msg.Offset, checkpoint.Offsets[key])
					continue
				}

				messages, offset, after := handler(msg)

				var metadata string
				if msg.Topic == h.TriggerTopic {
					if len(messages) > 0 {
						checkpoint.Set(key, msg.Offset+1)
					}

					metadata = checkpoint.Metadata()
				}

				transaction := &KafkaTransaction{
					Logger:    h.Logger,
					Producer:  h.Producer,
					GroupName: h.GroupName,
					Messages:  messages,
					Offset:    offset,
					Metadata:  metadata,
					After:     after,
				}

				func() {
					h.Lock()
					defer h.Unlock()
					if err := transaction.Commit(msg, session); err != nil {
						h.Logger.Errorw("Transaction error", zap.Error(err))
					}
				}()
			} else {
				// bump offset if we don't have a handler
				session.MarkMessage(msg, "")
				session.Commit()
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
