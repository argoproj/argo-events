package redisstream

import (
	"testing"

	redis "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

func TestWithoutURLDropsPassword(t *testing.T) {
	_, err := redis.ParseURL("redis://user:s3cret@localhost:6379/%zz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "s3cret", "the url.Parse error quotes the URL")

	err = withoutURL(err)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "s3cret")
}
