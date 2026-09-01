package base

import (
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMakeConnectionTLSOptional(t *testing.T) {
	logger := zap.NewNop().Sugar()

	t.Run("nil TLS does not require secure connection option before dial", func(t *testing.T) {
		js, err := NewJetstream("nats://127.0.0.1:1", "", &eventbuscommon.Auth{
			Strategy: v1alpha1.AuthStrategyNone,
		}, logger, nil)
		require.NoError(t, err)

		// Dial will fail because nothing is listening, but the important
		// part is we get a connection failure rather than a TLS handshake
		// error against a plain TCP endpoint.
		_, err = js.MakeConnection()
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "tls:")
		assert.NotContains(t, err.Error(), "TLS")
	})

	t.Run("configured TLS is still applied", func(t *testing.T) {
		js, err := NewJetstream("nats://127.0.0.1:1", "", &eventbuscommon.Auth{
			Strategy: v1alpha1.AuthStrategyNone,
		}, logger, &v1alpha1.TLSConfig{InsecureSkipVerify: true})
		require.NoError(t, err)

		_, err = js.MakeConnection()
		require.Error(t, err)
	})
}
