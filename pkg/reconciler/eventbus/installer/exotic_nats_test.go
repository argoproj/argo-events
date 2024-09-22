package installer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

const (
	testExoticName = "test-bus"
	testExoticURL  = "nats://xxxxxx"
)

var (
	testNatsExoticBus = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testExoticName,
		},
		Spec: v1alpha1.EventBusSpec{
			NATS: &v1alpha1.NATSBus{
				Exotic: &v1alpha1.NATSConfig{
					URL: testExoticURL,
				},
			},
		},
	}
)

func TestInstallationExotic(t *testing.T) {
	t.Run("installation with exotic nats config", func(t *testing.T) {
		installer := NewExoticNATSInstaller(testNatsExoticBus, logging.NewArgoEventsLogger())
		conf, err := installer.Install(context.TODO())
		assert.NoError(t, err)
		assert.NotNil(t, conf.NATS)
		assert.Equal(t, conf.NATS.URL, testExoticURL)
	})
}

func TestUninstallationExotic(t *testing.T) {
	t.Run("uninstallation with exotic nats config", func(t *testing.T) {
		installer := NewExoticNATSInstaller(testNatsExoticBus, logging.NewArgoEventsLogger())
		err := installer.Uninstall(context.TODO())
		assert.NoError(t, err)
	})
}
