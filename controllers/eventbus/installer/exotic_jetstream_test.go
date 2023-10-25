package installer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

var (
	testJSExoticURL = "nats://nats:4222"

	testJSExoticBus = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testExoticName,
		},
		Spec: v1alpha1.EventBusSpec{
			JetStreamExotic: &v1alpha1.JetStreamConfig{
				URL: testJSExoticURL,
			},
		},
	}
)

func TestInstallationJSExotic(t *testing.T) {
	t.Run("installation with exotic jetstream config", func(t *testing.T) {
		installer := NewExoticJetStreamInstaller(testJSExoticBus, logging.NewArgoEventsLogger())
		conf, err := installer.Install(context.TODO())
		assert.NoError(t, err)
		assert.NotNil(t, conf.JetStream)
		assert.Equal(t, conf.JetStream.URL, testJSExoticURL)
	})
}

func TestUninstallationJSExotic(t *testing.T) {
	t.Run("uninstallation with exotic jetstream config", func(t *testing.T) {
		installer := NewExoticJetStreamInstaller(testJSExoticBus, logging.NewArgoEventsLogger())
		err := installer.Uninstall(context.TODO())
		assert.NoError(t, err)
	})
}
