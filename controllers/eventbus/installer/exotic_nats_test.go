package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	testExoticName = "test-bus"
	testExoticURL  = "nats://xxxxxx"
)

var (
	testExoticBus = &v1alpha1.EventBus{
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
		installer := NewExoticNATSInstaller(testExoticBus, ctrl.Log.WithName("test"))
		conf, err := installer.Install()
		assert.NoError(t, err)
		assert.NotNil(t, conf.NATS)
		assert.Equal(t, conf.NATS.URL, testExoticURL)
	})
}

func TestUNinstallationExotic(t *testing.T) {
	t.Run("uninstallation with exotic nats config", func(t *testing.T) {
		installer := NewExoticNATSInstaller(testExoticBus, ctrl.Log.WithName("test"))
		err := installer.Uninstall()
		assert.NoError(t, err)
	})
}
