package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustHash(t *testing.T) {
	assert.Equal(t, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", MustHash([]byte("abc")))
	assert.Equal(t, "d4ffe8e9ee0b48eba716706123a7187f32eae3bdcb0e7763e41e533267bd8a53", MustHash("efg"))
	assert.Equal(t, "a8e084ec42eff43acd61526bef35e33ddf7a8135d6aba3b140a5cae4c8c5e10b", MustHash(
		struct {
			A string
			B string
		}{A: "aAa", B: "bBb"}))
}
