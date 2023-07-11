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
)

func TestNewSourceJetstream(t *testing.T) {
	logger := zap.NewExample().Sugar()

	auth := &common.Auth{}
	sourceJetstream, err := NewSourceJetstream(testURL, testEventSource, testStreamConfig, auth, logger)
	assert.NotNil(t, sourceJetstream)
	assert.Nil(t, err)
}
