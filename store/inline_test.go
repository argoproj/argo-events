package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInlineReader(t *testing.T) {
	myStr := "myStr"
	inlineReader, err := NewInlineReader(&myStr)
	assert.NotNil(t, inlineReader)
	assert.Nil(t, err)
	data, err := inlineReader.Read()
	assert.NotNil(t, data)
	assert.Nil(t, err)
	assert.Equal(t, string(data), myStr)
}
