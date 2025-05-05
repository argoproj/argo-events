package base

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestBrokers(t *testing.T) {
	config := &v1alpha1.KafkaBus{
		URL: "broker1:9092,broker2:9092",
	}

	logger := zap.NewNop().Sugar()
	kafka := NewKafka(config, logger)

	expectedBrokers := []string{"broker1:9092", "broker2:9092"}
	actualBrokers := kafka.Brokers()

	assert.Equal(t, expectedBrokers, actualBrokers)
}

func TestConfig(t *testing.T) {
	config := &v1alpha1.KafkaBus{
		URL: "localhost:9092",
	}

	logger := zap.NewNop().Sugar()

	kafka := NewKafka(config, logger)

	saramaConfig, err := kafka.Config()

	assert.NoError(t, err)
	assert.NotNil(t, saramaConfig)
	assert.Equal(t, sarama.OffsetNewest, saramaConfig.Consumer.Offsets.Initial)
	assert.Equal(t, sarama.WaitForAll, saramaConfig.Producer.RequiredAcks)
}

func TestConfig_StartOldest(t *testing.T) {
	config := &v1alpha1.KafkaBus{
		URL: "localhost:9092",
		ConsumerGroup: &v1alpha1.KafkaConsumerGroup{
			Oldest: true,
		},
	}

	logger := zap.NewNop().Sugar()

	kafka := NewKafka(config, logger)

	saramaConfig, err := kafka.Config()

	assert.NoError(t, err)
	assert.NotNil(t, saramaConfig)
	assert.Equal(t, sarama.OffsetOldest, saramaConfig.Consumer.Offsets.Initial)
}

func TestConfig_NoSASL(t *testing.T) {
	config := &v1alpha1.KafkaBus{
		URL:  "localhost:9092",
		SASL: nil,
	}

	logger := zap.NewNop().Sugar()

	kafka := NewKafka(config, logger)

	saramaConfig, err := kafka.Config()

	assert.NoError(t, err)
	assert.NotNil(t, saramaConfig)
	assert.False(t, saramaConfig.Net.SASL.Enable)
}

func TestConfig_TLS(t *testing.T) {
	config := &v1alpha1.KafkaBus{
		URL: "localhost:9092",
		TLS: &v1alpha1.TLSConfig{
			Enabled: true,
		},
	}

	logger := zap.NewNop().Sugar()

	kafka := NewKafka(config, logger)

	saramaConfig, err := kafka.Config()
	assert.NoError(t, err)
	assert.NotNil(t, kafka)
	assert.NotNil(t, kafka.Logger)
	assert.NotNil(t, kafka.config)
	assert.NotNil(t, saramaConfig)
	assert.True(t, saramaConfig.Net.TLS.Enable)
}

func TestNewKafka(t *testing.T) {
	config := &v1alpha1.KafkaBus{
		URL: "localhost:9092",
	}

	logger := zap.NewNop().Sugar()

	kafka := NewKafka(config, logger)

	assert.NotNil(t, kafka)
	assert.NotNil(t, kafka.Logger)
	assert.NotNil(t, kafka.config)
}

func TestNewKafka_EmptyURL(t *testing.T) {
	config := &v1alpha1.KafkaBus{
		URL: "",
	}

	logger := zap.NewNop().Sugar()

	kafka := NewKafka(config, logger)

	assert.NotNil(t, kafka)
	assert.NotNil(t, kafka.Logger)
	assert.NotNil(t, kafka.config)
}
