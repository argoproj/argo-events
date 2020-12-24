package persist

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
)

func TestConfigMapPersist(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	conf := &v1alpha1.ConfigMapPersistence{
		Name:             "test-config",
		CreateIfNotExist: true,
	}
	ctx := context.TODO()
	cp, err := NewConfigMapPersist(ctx, kubeClient, conf, "default")
	assert.NoError(t, err)
	assert.True(t, cp.IsEnabled())
	cm, err := kubeClient.CoreV1().ConfigMaps("default").Get(ctx, "test-config", v1.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, cm)
	assert.Nil(t, cm.Data)

	t.Run("SaveAndGetEvent", func(t *testing.T) {
		event := Event{
			EventKey:     "test.test",
			EventPayload: "{test}",
		}
		err = cp.Save(&event)
		assert.NoError(t, err)

		event1, err := cp.Get("test.test")
		assert.NoError(t, err)
		assert.NotNil(t, event1)
		assert.Equal(t, "test.test", event1.EventKey)

		event1, err = cp.Get("test.test1")
		assert.NoError(t, err)
		assert.Nil(t, event1)
	})

	t.Run("GetWithDeletedMap", func(t *testing.T) {
		err = kubeClient.CoreV1().ConfigMaps("default").Delete(ctx, "test-config", v1.DeleteOptions{})
		assert.NoError(t, err)
		event1, err := cp.Get("test.test")
		assert.NoError(t, err)
		assert.Nil(t, event1)
	})

	t.Run("SaveAndGetEventWithDeletedMap", func(t *testing.T) {
		err = kubeClient.CoreV1().ConfigMaps("default").Delete(ctx, "test-config", v1.DeleteOptions{})
		assert.True(t, apierr.IsNotFound(err))
		event := Event{
			EventKey:     "test.test",
			EventPayload: "{test}",
		}
		err = cp.Save(&event)
		assert.NoError(t, err)

		event1, err := cp.Get("test.test")
		assert.NoError(t, err)
		assert.NotNil(t, event1)
		assert.Equal(t, "test.test", event1.EventKey)

		event1, err = cp.Get("test.test1")
		assert.NoError(t, err)
		assert.Nil(t, event1)
	})
}
