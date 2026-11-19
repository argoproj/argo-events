package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

const sampleCreds = `-----BEGIN NATS USER JWT-----
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.ignored
------END NATS USER JWT------

-----BEGIN USER NKEY SEED-----
SUACSSL3UAYUDCMBHJOYFE3L6SWUH7C7UR6KJKFFDLSH7HK3ROHJQSMW34
------END USER NKEY SEED------
`

func TestIsNATSCredsFile(t *testing.T) {
	assert.True(t, IsNATSCredsFile([]byte(sampleCreds)))
	assert.False(t, IsNATSCredsFile([]byte("username: foo\npassword: bar\n")))
	assert.False(t, IsNATSCredsFile([]byte("BEGIN NATS USER JWT only")))
}

func TestLoadEventBusAuth(t *testing.T) {
	logger := zap.NewNop().Sugar()

	t.Run("none strategy", func(t *testing.T) {
		auth, err := LoadEventBusAuth("/unused", v1alpha1.AuthStrategyNone, logger, nil)
		require.NoError(t, err)
		assert.Equal(t, v1alpha1.AuthStrategyNone, auth.Strategy)
	})

	t.Run("yaml basic", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, v1alpha1.EventBusAuthFileName), []byte("username: u\npassword: p\n"), 0o600))
		auth, err := LoadEventBusAuth(dir, v1alpha1.AuthStrategyBasic, logger, nil)
		require.NoError(t, err)
		assert.Equal(t, v1alpha1.AuthStrategyBasic, auth.Strategy)
		assert.Equal(t, "u", auth.Credential.Username)
		assert.Equal(t, "p", auth.Credential.Password)
	})

	t.Run("creds from env", func(t *testing.T) {
		t.Setenv(v1alpha1.EnvVarEventBusNATSCredentials, sampleCreds)
		auth, err := LoadEventBusAuth("/unused", v1alpha1.AuthStrategyBasic, logger, nil)
		require.NoError(t, err)
		assert.Equal(t, v1alpha1.AuthStrategyCredential, auth.Strategy)
		assert.Equal(t, []byte(sampleCreds), auth.Credential.Credentials)
	})

	t.Run("yaml from env", func(t *testing.T) {
		t.Setenv(v1alpha1.EnvVarEventBusNATSCredentials, "username: u\npassword: p\n")
		auth, err := LoadEventBusAuth("/unused", v1alpha1.AuthStrategyBasic, logger, nil)
		require.NoError(t, err)
		assert.Equal(t, v1alpha1.AuthStrategyBasic, auth.Strategy)
		assert.Equal(t, "u", auth.Credential.Username)
		assert.Equal(t, "p", auth.Credential.Password)
	})

	t.Run("creds file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, v1alpha1.EventBusAuthFileName)
		require.NoError(t, os.WriteFile(path, []byte(sampleCreds), 0o600))
		auth, err := LoadEventBusAuth(dir, v1alpha1.AuthStrategyBasic, logger, nil)
		require.NoError(t, err)
		assert.Equal(t, v1alpha1.AuthStrategyCredential, auth.Strategy)
		assert.Equal(t, []byte(sampleCreds), auth.Credential.Credentials)
		assert.Empty(t, auth.Credential.Token)
	})

	t.Run("env preferred over file", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, v1alpha1.EventBusAuthFileName), []byte("username: file\npassword: file\n"), 0o600))
		t.Setenv(v1alpha1.EnvVarEventBusNATSCredentials, sampleCreds)
		auth, err := LoadEventBusAuth(dir, v1alpha1.AuthStrategyBasic, logger, nil)
		require.NoError(t, err)
		assert.Equal(t, v1alpha1.AuthStrategyCredential, auth.Strategy)
		assert.Equal(t, []byte(sampleCreds), auth.Credential.Credentials)
	})
}

func TestApplyNATSAuthOptions(t *testing.T) {
	t.Run("credential requires bytes", func(t *testing.T) {
		_, err := ApplyNATSAuthOptions(nil, &Auth{Strategy: v1alpha1.AuthStrategyCredential, Credential: &AuthCredential{}})
		require.Error(t, err)
	})

	t.Run("credential option", func(t *testing.T) {
		opts, err := ApplyNATSAuthOptions(nil, &Auth{
			Strategy:   v1alpha1.AuthStrategyCredential,
			Credential: &AuthCredential{Credentials: []byte(sampleCreds)},
		})
		require.NoError(t, err)
		assert.Len(t, opts, 1)
	})
}
