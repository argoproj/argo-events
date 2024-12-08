package eventsource

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
