package calendar

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalendarConfigExecutor_StopConfig(t *testing.T) {
	ce := &CalendarConfigExecutor{}
	ctx := &gateways.ConfigContext{}
	ctx.Active = true
	go func() {
		msg := <-ctx.StopChan
		assert.Equal(t, msg, struct{}{})
	}()
	ce.StopConfig(ctx)
	assert.Equal(t, false, ctx.Active)
}
