package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-events/common/logging"
)

func TestGetInstaller(t *testing.T) {
	t.Run("get installer", func(t *testing.T) {
		installer, err := getInstaller(testEventBus, nil, "", "", logging.NewArgoEventsLogger())
		assert.NoError(t, err)
		assert.NotNil(t, installer)
		_, ok := installer.(*natsInstaller)
		assert.True(t, ok)

		installer, err = getInstaller(testExoticBus, nil, "", "", logging.NewArgoEventsLogger())
		assert.NoError(t, err)
		assert.NotNil(t, installer)
		_, ok = installer.(*exoticNATSInstaller)
		assert.True(t, ok)
	})
}
