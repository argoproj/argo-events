package storagegrid

import (
	"github.com/argoproj/argo-events/gateways"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStorageGridConfigExecutor_StopConfig(t *testing.T) {
	s3Config := &StorageGridConfigExecutor{}
	ctx := &gateways.ConfigContext{}
	ctx.StopChan = make(chan struct{})
	ctx.Active = true
	go func() {
		msg := <-ctx.StopChan
		assert.Equal(t, msg, struct{}{})
	}()
	s3Config.StopConfig(ctx)
	assert.Equal(t, false, ctx.Active)
}
