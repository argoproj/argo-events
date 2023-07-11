package eventsource

import (
	"testing"

	"github.com/argoproj/argo-events/eventbus/common"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewSourceJetstream(t *testing.T) {
	logger := zap.NewExample().Sugar()
	defer logger.Sync()

	url := "test-url"
	eventSourceName := "test-event-source-name"
	streamConfig := "test-stream-config"
	auth := &common.Auth{}
	sourceJetstream, err := NewSourceJetstream(url, eventSourceName, streamConfig, auth, logger)
	assert.NotNil(t, sourceJetstream)
	assert.Nil(t, err)
}

func TestSourceJetstream_Initialize(t *testing.T) {
	logger := zap.NewExample().Sugar()
	defer logger.Sync()

	url := "test-url"
	eventSourceName := "test-event-source-name"
	streamConfig := "test-stream-config"
	auth := &common.Auth{}
	sourceJetstream, err := NewSourceJetstream(url, eventSourceName, streamConfig, auth, logger)
	sourceJetstream.Initialize()
	assert.NotNil(t, sourceJetstream)
	assert.Nil(t, err)
}

func TestSourceJetstream_Connect(t *testing.T) {
	logger := zap.NewExample().Sugar()
	defer logger.Sync()

	url := "test-url"
	eventSourceName := "test-event-source-name"
	streamConfig := "test-stream-config"
	auth := &common.Auth{}
	sourceJetstream, err := NewSourceJetstream(url, eventSourceName, streamConfig, auth, logger)
	sourceJetstream.Connect("test-client-id")
	assert.NotNil(t, sourceJetstream)
	assert.Nil(t, err)
}
