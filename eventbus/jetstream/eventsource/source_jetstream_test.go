package eventsource

import (
	"testing"

	"github.com/argoproj/argo-events/eventbus/common"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

const (
	testURL          = "test-url"
	testEventSource  = "test-event-source-name"
	testStreamConfig = "test-stream-config"
)

func TestNewSourceJetstream(t *testing.T) {
	logger := zap.NewExample().Sugar()

	auth := &common.Auth{}
	sourceJetstream, err := NewSourceJetstream(testURL, testEventSource, testStreamConfig, auth, logger)
	assert.NotNil(t, sourceJetstream)
	assert.Nil(t, err)
}

func TestSourceJetstream_Connect(t *testing.T) {
	logger := zap.NewExample().Sugar()

	auth := &common.Auth{}
	sourceJetstream, err := NewSourceJetstream(testURL, testEventSource, testStreamConfig, auth, logger)
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
	sourceJetstream, err := NewSourceJetstream(testURL, testEventSource, testStreamConfig, auth, logger)
	assert.NotNil(t, sourceJetstream)
	assert.Nil(t, err)

	err = sourceJetstream.Initialize()
	assert.NotNil(t, err)
}
