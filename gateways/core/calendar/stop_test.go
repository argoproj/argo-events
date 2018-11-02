package calendar

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestCalendarConfigExecutor_StopConfig(t *testing.T) {
	ce := &CalendarConfigExecutor{}
	ctx := getConfigContext()
	ctx.Active = true
	go func() {
		msg :=<- ctx.StopCh
		assert.Equal(t, msg, struct {}{})
	}()
	err := ce.StopConfig(ctx)
	assert.Nil(t, err)
}
