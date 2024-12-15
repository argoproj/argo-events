package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomString(t *testing.T) {
	str := RandomString(20)
	assert.Equal(t, 20, len(str))
}

func TestConvertToDNSLabel(t *testing.T) {
	str := "_123Th-123__2343Test-"
	assert.Equal(t, "123th-123--2343test", ConvertToDNSLabel(str))
}
