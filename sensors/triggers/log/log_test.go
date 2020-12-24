package log

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"

	sv1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func TestLogTrigger(t *testing.T) {
	logger := zaptest.NewLogger(t)
	l := &LogTrigger{
		Logger:  logger,
		Trigger: &sv1.Trigger{Template: &sv1.TriggerTemplate{}},
	}
	_, err := l.Execute(context.TODO(), map[string]*sv1.Event{"my-event": {}}, &sv1.LogTrigger{})
	assert.NoError(t, err)

	assert.True(t, l.shouldLog(&sv1.LogTrigger{}))
	assert.False(t, l.shouldLog(&sv1.LogTrigger{IntervalSeconds: 1}))

	time.Sleep(1 * time.Second)

	assert.True(t, l.shouldLog(&sv1.LogTrigger{}))
	assert.True(t, l.shouldLog(&sv1.LogTrigger{IntervalSeconds: 1}))
}
