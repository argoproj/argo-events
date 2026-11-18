package sensor

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	nats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	eventbuscommon "github.com/argoproj/argo-events/pkg/eventbus/common"
)

func startJetStreamServer(t *testing.T, storeDir string, port int) *server.Server {
	t.Helper()
	natsServer, err := server.NewServer(&server.Options{
		JetStream: true,
		StoreDir:  storeDir,
		Host:      "127.0.0.1",
		Port:      port,
	})
	require.NoError(t, err)
	natsServer.Start()
	t.Cleanup(func() {
		natsServer.Shutdown()
		natsServer.WaitForShutdown()
	})
	require.True(t, natsServer.ReadyForConnections(10*time.Second))
	return natsServer
}

func TestSensorJetstreamConnectClosesConnectionOnTriggerConnectionError(t *testing.T) {
	natsServer := startJetStreamServer(t, t.TempDir(), -1)

	sensorSpec := &v1alpha1.Sensor{ObjectMeta: metav1.ObjectMeta{Name: "invalid-expression"}}
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

	triggerConn, err := stream.Connect(context.Background(), "trigger", "(", nil, false)
	require.Error(t, err)
	require.Nil(t, triggerConn)
	require.ErrorContains(t, err, "failed to evaluate expression")
	require.Eventually(t, func() bool {
		return natsServer.NumClients() == baseline
	}, 5*time.Second, 10*time.Millisecond)
}

func TestSensorJetstreamConnectRecreatesStreamAndKeyValueStore(t *testing.T) {
	storeDir := t.TempDir()
	natsServer := startJetStreamServer(t, storeDir, -1)
	port := natsServer.Addr().(*net.TCPAddr).Port

	sensorSpec := &v1alpha1.Sensor{ObjectMeta: metav1.ObjectMeta{Name: "lost-eventbus-state"}}
	stream, err := NewSensorJetstream(
		natsServer.ClientURL(),
		sensorSpec,
		"",
		&eventbuscommon.Auth{Strategy: v1alpha1.AuthStrategyNone},
		zap.NewNop().Sugar(),
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, stream.Initialize())

	// the Stream and the K/V store already exist
	triggerConn, err := stream.Connect(context.Background(), "trigger", "dependency", nil, false)
	require.NoError(t, err)
	require.NoError(t, triggerConn.Close())

	// the EventBus comes back with an empty store
	natsServer.Shutdown()
	natsServer.WaitForShutdown()
	require.NoError(t, os.RemoveAll(storeDir))
	natsServer = startJetStreamServer(t, storeDir, port)

	triggerConn, err = stream.Connect(context.Background(), "trigger", "dependency", nil, false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = triggerConn.Close() })

	nc, err := nats.Connect(natsServer.ClientURL())
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)
	_, err = js.StreamInfo(v1alpha1.JetStreamStreamName)
	require.NoError(t, err)
	_, err = js.KeyValue(sensorSpec.Name)
	require.NoError(t, err)
}
