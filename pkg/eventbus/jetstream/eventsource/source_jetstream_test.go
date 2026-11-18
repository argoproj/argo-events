package eventsource

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	nats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/eventbus/common"
)

const (
	testURL          = "test-url"
	testEventSource  = "test-event-source-name"
	testStreamConfig = "test-stream-config"
)

func TestNewSourceJetstream(t *testing.T) {
	logger := zap.NewExample().Sugar()

	auth := &common.Auth{}
	sourceJetstream, err := NewSourceJetstream(testURL, testEventSource, testStreamConfig, auth, logger, nil)
	assert.NotNil(t, sourceJetstream)
	assert.Nil(t, err)
}

func TestSourceJetstream_Connect(t *testing.T) {
	logger := zap.NewExample().Sugar()

	auth := &common.Auth{}
	sourceJetstream, err := NewSourceJetstream(testURL, testEventSource, testStreamConfig, auth, logger, nil)
	assert.NotNil(t, sourceJetstream)
	assert.Nil(t, err)

	conn, err := sourceJetstream.Connect("test-client-id")
	assert.Nil(t, conn)
	assert.NotNil(t, err)
}

func TestSourceJetstream_Initialize_Failure(t *testing.T) {
	logger := zap.NewExample().Sugar()

	auth := &common.Auth{
		Strategy: v1alpha1.AuthStrategyNone,
	}
	sourceJetstream, err := NewSourceJetstream(testURL, testEventSource, testStreamConfig, auth, logger, nil)
	assert.NotNil(t, sourceJetstream)
	assert.Nil(t, err)

	err = sourceJetstream.Initialize()
	assert.NotNil(t, err)
}

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

func TestSourceJetstreamConnectRecreatesStream(t *testing.T) {
	storeDir := t.TempDir()
	natsServer := startJetStreamServer(t, storeDir, -1)
	port := natsServer.Addr().(*net.TCPAddr).Port

	sourceJetstream, err := NewSourceJetstream(
		natsServer.ClientURL(),
		testEventSource,
		"",
		&common.Auth{Strategy: v1alpha1.AuthStrategyNone},
		zap.NewNop().Sugar(),
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, sourceJetstream.Initialize())

	msg := common.Message{
		MsgHeader: common.MsgHeader{EventSourceName: testEventSource, EventName: "test-event", ID: "msg-1"},
		Body:      []byte(`{"hello":"world"}`),
	}

	// the Stream already exists
	conn, err := sourceJetstream.Connect("test-client-id")
	require.NoError(t, err)
	require.NoError(t, conn.Publish(context.Background(), msg))
	require.NoError(t, conn.Close())

	// the EventBus comes back with an empty store
	natsServer.Shutdown()
	natsServer.WaitForShutdown()
	require.NoError(t, os.RemoveAll(storeDir))
	natsServer = startJetStreamServer(t, storeDir, port)

	conn, err = sourceJetstream.Connect("test-client-id")
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	msg.ID = "msg-2"
	require.NoError(t, conn.Publish(context.Background(), msg))

	nc, err := nats.Connect(natsServer.ClientURL())
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := nc.JetStream()
	require.NoError(t, err)
	_, err = js.StreamInfo(v1alpha1.JetStreamStreamName)
	require.NoError(t, err)
}
