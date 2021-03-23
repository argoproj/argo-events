package github

import (
	"testing"

	gh "github.com/google/go-github/v31/github"
	"github.com/stretchr/testify/assert"
)

func TestSliceEqual(t *testing.T) {
	assert.True(t, sliceEqual(nil, nil))
	assert.True(t, sliceEqual([]string{"hello"}, []string{"hello"}))
	assert.True(t, sliceEqual([]string{"hello", "world"}, []string{"hello", "world"}))
	assert.True(t, sliceEqual([]string{}, []string{}))

	assert.False(t, sliceEqual([]string{"hello"}, nil))
	assert.False(t, sliceEqual([]string{"hello"}, []string{}))
	assert.False(t, sliceEqual([]string{}, []string{"hello"}))
	assert.False(t, sliceEqual([]string{"hello"}, []string{"hello", "world"}))
	assert.False(t, sliceEqual([]string{"hello", "world"}, []string{"hello"}))
	assert.False(t, sliceEqual([]string{"hello", "world"}, []string{"hello", "moon"}))
	assert.True(t, sliceEqual([]string{"hello", "world"}, []string{"world", "hello"}))
	assert.True(t, sliceEqual([]string{"hello", "*"}, []string{"*"}))
	assert.True(t, sliceEqual([]string{"hello", "*"}, []string{"*", "world"}))
	assert.True(t, sliceEqual([]string{"hello", "world", "hello"}, []string{"hello", "hello", "world", "world"}))
	assert.True(t, sliceEqual([]string{"world", "hello"}, []string{"hello", "hello", "world", "world"}))
	assert.True(t, sliceEqual([]string{"hello", "hello", "world", "world"}, []string{"world", "hello"}))
	assert.False(t, sliceEqual([]string{"hello"}, []string{"*", "hello"}))
	assert.False(t, sliceEqual([]string{"hello", "*"}, []string{"hello"}))
	assert.False(t, sliceEqual([]string{"*", "hello", "*"}, []string{"hello"}))
	assert.False(t, sliceEqual([]string{"hello"}, []string{"world", "world"}))
	assert.False(t, sliceEqual([]string{"hello", "hello"}, []string{"world", "world"}))
	assert.True(t, sliceEqual([]string{"*", "hello", "*"}, []string{"*", "world", "hello", "world"}))
}

func TestCompareHook(t *testing.T) {
	assert.False(t, compareHook(nil, "https://google.com/", []string{}))

	assert.True(t, compareHook(&gh.Hook{
		Config: map[string]interface{}{
			"url": "https://google.com/",
		},
		Events: []string{"*"},
	}, "https://google.com/", []string{"*"}))

	assert.False(t, compareHook(&gh.Hook{
		Config: map[string]interface{}{
			"url": "https://google.com/",
		},
		Events: []string{"pull_request"},
	}, "https://google.com/", []string{"*"}))

	assert.False(t, compareHook(&gh.Hook{
		Config: map[string]interface{}{
			"url": "https://example.com/",
		},
		Events: []string{"pull_request"},
	}, "https://google.com/", []string{"*"}))
}

func TestGetHook(t *testing.T) {
	hooks := []*gh.Hook{
		{
			Config: map[string]interface{}{
				"url": "https://example.com/",
			},
			Events: []string{"pull_request"},
		},
		{
			Config: map[string]interface{}{
				"url": "https://example.com/",
			},
			Events: []string{"*"},
		},
	}

	assert.Equal(t, hooks[1], getHook(hooks, "https://example.com/", []string{"*"}))
	assert.Nil(t, getHook(hooks, "https://example.com/", []string{"does_not_exist"}))
}
