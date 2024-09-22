package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomString(t *testing.T) {
	str := RandomString(20)
	assert.Equal(t, 20, len(str))
}
