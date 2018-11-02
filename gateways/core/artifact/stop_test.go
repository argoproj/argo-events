package artifact

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestS3ConfigExecutor_StopConfig(t *testing.T) {
	s3Config := &S3ConfigExecutor{}
	ctx := getConfigContext()
	ctx.Active = true
	go func() {
		msg :=<- ctx.StopCh
		assert.Equal(t, msg, struct {}{})
	}()
	err := s3Config.StopConfig(ctx)
	assert.Nil(t, err)
}
