package github

import (
	"testing"

	gh "github.com/google/go-github/v50/github"
	"github.com/stretchr/testify/assert"
)

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
