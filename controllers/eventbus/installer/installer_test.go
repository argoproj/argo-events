package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestGetInstaller(t *testing.T) {
	t.Run("get installer", func(t *testing.T) {
		installer, err := getInstaller(testEventBus, nil, "", ctrl.Log.WithName("test"))
		assert.NoError(t, err)
		assert.NotNil(t, installer)
		_, ok := installer.(*natsInstaller)
		assert.True(t, ok)

		installer, err = getInstaller(testExoticBus, nil, "", ctrl.Log.WithName("test"))
		assert.NoError(t, err)
		assert.NotNil(t, installer)
		_, ok = installer.(*exoticNATSInstaller)
		assert.True(t, ok)
	})
}
