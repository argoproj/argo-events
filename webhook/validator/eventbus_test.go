package validator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

func TestValidateEventBusDelete(t *testing.T) {
	eb := fakeEventBus()
	es := fakeCalendarEventSource()
	t.Run("test delete event bus without eventsource and sensor connected", func(t *testing.T) {
		_, err := fakeEventBusClient.ArgoprojV1alpha1().EventBus(testNamespace).Create(context.Background(), eb, metav1.CreateOptions{})
		assert.NoError(t, err)
		v := NewEventBusValidator(fakeK8sClient, fakeEventBusClient, fakeEventSourceClient, fakeSensorClient, eb, nil)
		r := v.ValidateDelete(contextWithLogger(t))
		assert.True(t, r.Allowed)
		err = fakeEventBusClient.ArgoprojV1alpha1().EventBus(testNamespace).Delete(context.Background(), eb.Name, metav1.DeleteOptions{})
		assert.NoError(t, err)
	})

	t.Run("test delete event bus with eventsource connected", func(t *testing.T) {
		_, err := fakeEventBusClient.ArgoprojV1alpha1().EventBus(testNamespace).Create(context.Background(), eb, metav1.CreateOptions{})
		assert.NoError(t, err)
		_, err = fakeEventSourceClient.ArgoprojV1alpha1().EventSources(testNamespace).Create(context.Background(), es, metav1.CreateOptions{})
		assert.NoError(t, err)
		v := NewEventBusValidator(fakeK8sClient, fakeEventBusClient, fakeEventSourceClient, fakeSensorClient, eb, nil)
		r := v.ValidateDelete(contextWithLogger(t))
		assert.False(t, r.Allowed)
		err = fakeEventSourceClient.ArgoprojV1alpha1().EventSources(testNamespace).Delete(context.Background(), es.Name, metav1.DeleteOptions{})
		assert.NoError(t, err)
		r = v.ValidateDelete(contextWithLogger(t))
		assert.True(t, r.Allowed)
		err = fakeEventBusClient.ArgoprojV1alpha1().EventBus(testNamespace).Delete(context.Background(), eb.Name, metav1.DeleteOptions{})
		assert.NoError(t, err)
	})
}
