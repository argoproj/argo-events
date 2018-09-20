package gateways

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func Test_transformPayload(t *testing.T) {
	payload := []byte("hello")
	src := "test"
	_, err := TransformerPayload(payload, src)
	assert.Nil(t, err)
}