package kafka

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// fakeConsumerGroupSession is a minimal sarama.ConsumerGroupSession whose
// Context() stays live for the lifetime of the test, mirroring a session
// that has not been rebalanced away.
type fakeConsumerGroupSession struct {
	ctx context.Context
}

func (f *fakeConsumerGroupSession) Claims() map[string][]int32                  { return nil }
func (f *fakeConsumerGroupSession) MemberID() string                            { return "fake-member" }
func (f *fakeConsumerGroupSession) GenerationID() int32                         { return 1 }
func (f *fakeConsumerGroupSession) MarkOffset(string, int32, int64, string)     {}
func (f *fakeConsumerGroupSession) Commit()                                     {}
func (f *fakeConsumerGroupSession) ResetOffset(string, int32, int64, string)    {}
func (f *fakeConsumerGroupSession) MarkMessage(*sarama.ConsumerMessage, string) {}
func (f *fakeConsumerGroupSession) Context() context.Context                    { return f.ctx }

// fakeConsumerGroupClaim exposes a message channel the test controls
// directly, so it can be closed to simulate a sarama rebalance.
type fakeConsumerGroupClaim struct {
	topic     string
	partition int32
	messages  chan *sarama.ConsumerMessage
}

func (f *fakeConsumerGroupClaim) Topic() string                            { return f.topic }
func (f *fakeConsumerGroupClaim) Partition() int32                         { return f.partition }
func (f *fakeConsumerGroupClaim) InitialOffset() int64                     { return 0 }
func (f *fakeConsumerGroupClaim) HighWaterMarkOffset() int64               { return 0 }
func (f *fakeConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage { return f.messages }

// TestConsumeClaim_ReturnsOnClosedChannel reproduces the DEV-209308 hang: a
// Kafka broker restart closes the claim's message channel while the
// session's context is still alive. ConsumeClaim must return so sarama can
// rebalance, instead of looping forever on the nil slice a closed channel
// produces.
func TestConsumeClaim_ReturnsOnClosedChannel(t *testing.T) {
	const topic = "test-topic"
	const partition = int32(0)

	claim := &fakeConsumerGroupClaim{
		topic:     topic,
		partition: partition,
		messages:  make(chan *sarama.ConsumerMessage),
	}
	session := &fakeConsumerGroupSession{ctx: context.Background()}

	h := &KafkaHandler{
		Mutex:  &sync.Mutex{},
		Logger: zap.NewNop().Sugar(),
		Handlers: map[string]func(*sarama.ConsumerMessage) ([]*sarama.ProducerMessage, int64, func()){
			topic: func(*sarama.ConsumerMessage) ([]*sarama.ProducerMessage, int64, func()) {
				return nil, 0, nil
			},
		},
		checkpoints: Checkpoints{
			topic: {
				partition: &Checkpoint{Logger: zap.NewNop().Sugar()},
			},
		},
	}

	// Close the claim's channel to simulate the broker-restart rebalance,
	// while the session context stays alive (its Done() never fires).
	close(claim.messages)

	done := make(chan error, 1)
	go func() {
		done <- h.ConsumeClaim(session, claim)
	}()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("ConsumeClaim did not return after its message channel closed; it is stuck in the hot-loop from DEV-209308")
	}
}
