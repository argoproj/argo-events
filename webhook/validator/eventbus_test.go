package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

func TestValidateEventBusCreate(t *testing.T) {
	eb := fakeEventBus()
	v := NewEventBusValidator(fakeK8sClient, fakeEventBusClient, fakeEventSourceClient, fakeSensorClient, nil, eb)
	r := v.ValidateCreate(contextWithLogger(t))
	assert.True(t, r.Allowed)
}

func TestValidateEventBusUpdate(t *testing.T) {
	eb := fakeEventBus()
	t.Run("test update auth strategy", func(t *testing.T) {
		newEb := eb.DeepCopy()
		newEb.Generation++
		newEb.Spec.NATS.Native.Auth = nil
		v := NewEventBusValidator(fakeK8sClient, fakeEventBusClient, fakeEventSourceClient, fakeSensorClient, eb, newEb)
		r := v.ValidateUpdate(contextWithLogger(t))
		assert.False(t, r.Allowed)
	})

	t.Run("test update to exotic", func(t *testing.T) {
		newEb := eb.DeepCopy()
		newEb.Generation++
		newEb.Spec.NATS.Native = nil
		cID := "test-id"
		newEb.Spec.NATS.Exotic = &eventbusv1alpha1.NATSConfig{
			ClusterID: &cID,
			URL:       "nats://abc:1234",
		}
		v := NewEventBusValidator(fakeK8sClient, fakeEventBusClient, fakeEventSourceClient, fakeSensorClient, eb, newEb)
		r := v.ValidateUpdate(contextWithLogger(t))
		assert.False(t, r.Allowed)
	})

	t.Run("test update to native", func(t *testing.T) {
		exoticEb := fakeExoticEventBus()
		newEb := exoticEb.DeepCopy()
		newEb.Generation++
		newEb.Spec.NATS.Exotic = nil
		newEb.Spec.NATS.Native = eb.Spec.NATS.Native
		v := NewEventBusValidator(fakeK8sClient, fakeEventBusClient, fakeEventSourceClient, fakeSensorClient, exoticEb, newEb)
		r := v.ValidateUpdate(contextWithLogger(t))
		assert.False(t, r.Allowed)
	})
}
