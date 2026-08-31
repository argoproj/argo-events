package sensor

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
)

func TestSensorJetstreamConnectClosesConnectionOnTriggerConnectionError(t *testing.T) {
	natsServer, err := server.NewServer(&server.Options{
		JetStream: true,
		StoreDir:  t.TempDir(),
		Host:      "127.0.0.1",
		Port:      -1,
	})
	require.NoError(t, err)
	natsServer.Start()
	t.Cleanup(func() {
		natsServer.Shutdown()
		natsServer.WaitForShutdown()
	})
	require.True(t, natsServer.ReadyForConnections(10*time.Second))

	sensorSpec := &v1alpha1.Sensor{ObjectMeta: metav1.ObjectMeta{Name: "missing-kv-bucket"}}
	stream, err := NewSensorJetstream(
		natsServer.ClientURL(),
		sensorSpec,
		"",
		&eventbuscommon.Auth{Strategy: v1alpha1.AuthStrategyNone},
		zap.NewNop().Sugar(),
		nil,
	)
	require.NoError(t, err)
	baseline := natsServer.NumClients()

	triggerConn, err := stream.Connect(context.Background(), "trigger", "dependency", nil, false)
	require.Error(t, err)
	require.Nil(t, triggerConn)
	require.ErrorContains(t, err, "failed to get K/V store")
	require.Eventually(t, func() bool {
		return natsServer.NumClients() == baseline
	}, 5*time.Second, 10*time.Millisecond)
}
