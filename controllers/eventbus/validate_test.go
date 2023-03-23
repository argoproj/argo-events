package eventbus

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

var (
	testNatsEventBus = &v1alpha1.EventBus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      common.DefaultEventBusName,
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
			Namespace: "test-ns",
			Name:      common.DefaultEventBusName,
		},
		Spec: v1alpha1.EventBusSpec{
			JetStream: &v1alpha1.JetStreamBus{
				Version: "2.7.3",
			},
		},
	}

	testKafkaEventBus = &v1alpha1.EventBus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      common.DefaultEventBusName,
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

	t.Run("test js eventbus replica", func(t *testing.T) {
		eb := testJetStreamEventBus.DeepCopy()
		eb.Spec.JetStream.Replicas = pointer.Int32(2)
		err := ValidateEventBus(eb)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid spec: a jetstream eventbus requires at least 3 replicas")
		eb.Spec.JetStream.Replicas = pointer.Int32(3)
		err = ValidateEventBus(eb)
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
		assert.Contains(t, err.Error(), "\"spec.kafka.url\" is missing")
	})
}
