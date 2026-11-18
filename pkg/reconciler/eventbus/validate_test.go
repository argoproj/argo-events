package eventbus

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

var (
	testNatsEventBus = &v1alpha1.EventBus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      v1alpha1.DefaultEventBusName,
		},
		Spec: v1alpha1.EventBusSpec{
			NATS: &v1alpha1.NATSBus{
				Native: &v1alpha1.NativeStrategy{
					Auth: &v1alpha1.AuthStrategyToken,
				},
			},
		},
	}

	testJetStreamEventBus = &v1alpha1.EventBus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      v1alpha1.DefaultEventBusName,
		},
		Spec: v1alpha1.EventBusSpec{
			JetStream: &v1alpha1.JetStreamBus{
				Version: "2.7.3",
			},
		},
	}

	testJetStreamExoticBus = &v1alpha1.EventBus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      v1alpha1.DefaultEventBusName,
		},
		Spec: v1alpha1.EventBusSpec{
			JetStreamExotic: &v1alpha1.JetStreamConfig{
				URL: "nats://nats:4222",
			},
		},
	}

	testKafkaEventBus = &v1alpha1.EventBus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      v1alpha1.DefaultEventBusName,
		},
		Spec: v1alpha1.EventBusSpec{
			Kafka: &v1alpha1.KafkaBus{
				URL: "127.0.0.1:9092",
			},
		},
	}
)

func TestValidate(t *testing.T) {
	t.Run("test good nats eventbus", func(t *testing.T) {
		err := ValidateEventBus(testNatsEventBus)
		assert.NoError(t, err)
	})

	t.Run("test good js eventbus", func(t *testing.T) {
		err := ValidateEventBus(testJetStreamEventBus)
		assert.NoError(t, err)
	})

	t.Run("test good kafka eventbus", func(t *testing.T) {
		err := ValidateEventBus(testKafkaEventBus)
		assert.NoError(t, err)
	})

	t.Run("test good js exotic eventbus", func(t *testing.T) {
		err := ValidateEventBus(testJetStreamExoticBus)
		assert.NoError(t, err)
	})

	t.Run("test bad eventbus", func(t *testing.T) {
		eb := testNatsEventBus.DeepCopy()
		eb.Spec.NATS = nil
		err := ValidateEventBus(eb)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid spec: either")
	})

	t.Run("test native nats exotic conflicting eventbus", func(t *testing.T) {
		eb := testNatsEventBus.DeepCopy()
		eb.Spec.NATS.Exotic = &v1alpha1.NATSConfig{}
		err := ValidateEventBus(eb)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "can not be defined together"))
	})

	t.Run("test exotic nats eventbus no clusterID", func(t *testing.T) {
		eb := testNatsEventBus.DeepCopy()
		eb.Spec.NATS.Native = nil
		eb.Spec.NATS.Exotic = &v1alpha1.NATSConfig{}
		err := ValidateEventBus(eb)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "\"spec.nats.exotic.clusterID\" is missing"))
	})

	t.Run("test exotic nats eventbus empty URL", func(t *testing.T) {
		eb := testNatsEventBus.DeepCopy()
		eb.Spec.NATS.Native = nil
		cID := "test-cluster-id"
		eb.Spec.NATS.Exotic = &v1alpha1.NATSConfig{
			ClusterID: &cID,
		}
		err := ValidateEventBus(eb)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "\"spec.nats.exotic.url\" is missing"))
	})

	t.Run("test js eventbus no version", func(t *testing.T) {
		eb := testJetStreamEventBus.DeepCopy()
		eb.Spec.JetStream.Version = ""
		err := ValidateEventBus(eb)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid spec: a version")
	})

	t.Run("test js eventbus client advertise", func(t *testing.T) {
		for _, v := range []string{"my-lb.elb.amazonaws.com:4222", "10.0.0.1:4222", "[2001:db8::1]:4222"} {
			eb := testJetStreamEventBus.DeepCopy()
			eb.Spec.JetStream.ClientAdvertise = ptr.To(v)
			assert.NoError(t, ValidateEventBus(eb), v)
		}
		for _, v := range []string{"", "my-lb", ":4222", "my-lb:", "my-lb:abc", "my-lb:70000", "nats://my-lb:4222", "my-lb:4222\"", "my lb:4222"} {
			eb := testJetStreamEventBus.DeepCopy()
			eb.Spec.JetStream.ClientAdvertise = ptr.To(v)
			err := ValidateEventBus(eb)
			assert.Error(t, err, v)
			if err != nil {
				assert.Contains(t, err.Error(), "spec.jetstream.clientAdvertise", v)
			}
		}
	})

	t.Run("test js eventbus replica", func(t *testing.T) {
		eb := testJetStreamEventBus.DeepCopy()
		eb.Spec.JetStream.Replicas = ptr.To[int32](3)
		err := ValidateEventBus(eb)
		assert.NoError(t, err)
		eb.Spec.JetStream.Replicas = nil
		err = ValidateEventBus(eb)
		assert.NoError(t, err)
	})

	t.Run("test kafka eventbus no URL", func(t *testing.T) {
		eb := testKafkaEventBus.DeepCopy()
		eb.Spec.Kafka.URL = ""
		err := ValidateEventBus(eb)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "either \"spec.kafka.url\" or \"spec.kafka.urlSecret\" is required")
	})

	t.Run("test kafka eventbus with URL secret", func(t *testing.T) {
		eb := testKafkaEventBus.DeepCopy()
		eb.Spec.Kafka.URL = ""
		eb.Spec.Kafka.URLSecret = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "kafka"},
			Key:                  "url",
		}
		assert.NoError(t, ValidateEventBus(eb))
	})

	t.Run("test kafka eventbus with both URL and URL secret", func(t *testing.T) {
		eb := testKafkaEventBus.DeepCopy()
		eb.Spec.Kafka.URLSecret = &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "kafka"},
			Key:                  "url",
		}
		err := ValidateEventBus(eb)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "can't both be set")
	})

	t.Run("test exotic js eventbus empty URL", func(t *testing.T) {
		eb := testJetStreamExoticBus.DeepCopy()
		eb.Spec.JetStreamExotic.URL = ""
		err := ValidateEventBus(eb)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "\"spec.jetstreamExotic.url\" is missing"))
	})
}
