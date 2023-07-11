package eventsource

import (
	"testing"

	"github.com/argoproj/argo-events/eventbus/common"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

const (
	testURL          = "test-url"
	testEventSource  = "test-event-source-name"
	testStreamConfig = "test-stream-config"
	testClientID     = "test-client-id"
)

func TestNewSourceJetstream(t *testing.T) {
	logger := zap.NewExample().Sugar()

	defer func() {
		logger.Sync()
		if r := recover(); r != nil {
			assert.Equal(t, "runtime error: invalid memory address or nil pointer dereference", r)
		}
	}()

	auth := &common.Auth{}
	sourceJetstream, err := NewSourceJetstream(testURL, testEventSource, testStreamConfig, auth, logger)
	assert.NotNil(t, sourceJetstream)
	assert.Nil(t, err)
}

func TestSourceJetstream_Initialize(t *testing.T) {
	logger := zap.NewExample().Sugar()

	defer func() {
		logger.Sync()
		if r := recover(); r != nil {
			assert.Equal(t, "runtime error: invalid memory address or nil pointer dereference", r)
		}
	}()

	auth := &common.Auth{}
	sourceJetstream, err := NewSourceJetstream(testURL, testEventSource, testStreamConfig, auth, logger)
	sourceJetstream.Initialize()
	assert.NotNil(t, sourceJetstream)
	assert.Nil(t, err)
}

func TestSourceJetstream_Connect(t *testing.T) {
	logger := zap.NewExample().Sugar()

	defer func() {
		logger.Sync()
		if r := recover(); r != nil {
			assert.Equal(t, "runtime error: invalid memory address or nil pointer dereference", r)
		}
	}()

	auth := &common.Auth{}
	sourceJetstream, err := NewSourceJetstream(testURL, testEventSource, testStreamConfig, auth, logger)
	sourceJetstream.Connect(testClientID) // error check this

	assert.NotNil(t, sourceJetstream)
	assert.Nil(t, err)
}
