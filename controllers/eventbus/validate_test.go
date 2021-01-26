package eventbus

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

var (
	testEventBus = &v1alpha1.EventBus{
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
)

func TestValidate(t *testing.T) {
	t.Run("test good eventbus", func(t *testing.T) {
		err := ValidateEventBus(testEventBus)
		assert.NoError(t, err)
	})

	t.Run("test native exotic conflicting eventbus", func(t *testing.T) {
		eb := testEventBus.DeepCopy()
		eb.Spec.NATS.Exotic = &v1alpha1.NATSConfig{}
		err := ValidateEventBus(eb)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "can not be defined together"))
	})

	t.Run("test exotic eventbus no clusterID", func(t *testing.T) {
		eb := testEventBus.DeepCopy()
		eb.Spec.NATS.Native = nil
		eb.Spec.NATS.Exotic = &v1alpha1.NATSConfig{}
		err := ValidateEventBus(eb)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "\"spec.nats.exotic.clusterID\" is missing"))
	})

	t.Run("test exotic eventbus empty URL", func(t *testing.T) {
		eb := testEventBus.DeepCopy()
		eb.Spec.NATS.Native = nil
		cID := "test-cluster-id"
		eb.Spec.NATS.Exotic = &v1alpha1.NATSConfig{
			ClusterID: &cID,
		}
		err := ValidateEventBus(eb)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "\"spec.nats.exotic.url\" is missing"))
	})
}
