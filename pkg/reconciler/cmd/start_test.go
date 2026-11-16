package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurationFromEnv(t *testing.T) {
	const env = "ARGO_EVENTS_TEST_DURATION"

	d, err := durationFromEnv(env)
	require.NoError(t, err)
	assert.Nil(t, d, "unset variable keeps the default")

	t.Setenv(env, "")
	d, err = durationFromEnv(env)
	require.NoError(t, err)
	assert.Nil(t, d, "empty variable keeps the default")

	t.Setenv(env, "30s")
	d, err = durationFromEnv(env)
	require.NoError(t, err)
	require.NotNil(t, d)
	assert.Equal(t, 30*time.Second, *d)

	for _, bad := range []string{"30", "soon", "0s", "-5s"} {
		t.Setenv(env, bad)
		_, err = durationFromEnv(env)
		assert.ErrorContains(t, err, env, bad)
	}
}
