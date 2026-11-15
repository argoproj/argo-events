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
			testURLKey: "https://google.com/",
		},
		Events: []string{"*"},
	}, "https://google.com/", []string{"*"}))

	assert.False(t, compareHook(&gh.Hook{
		Config: map[string]interface{}{
			testURLKey: "https://google.com/",
		},
		Events: []string{testPullRequestEvent},
	}, "https://google.com/", []string{"*"}))

	assert.False(t, compareHook(&gh.Hook{
		Config: map[string]interface{}{
			testURLKey: testHookURL,
		},
		Events: []string{testPullRequestEvent},
	}, "https://google.com/", []string{"*"}))
}

func TestGetHook(t *testing.T) {
	hooks := []*gh.Hook{
		{
			Config: map[string]interface{}{
				testURLKey: testHookURL,
			},
			Events: []string{testPullRequestEvent},
		},
		{
			Config: map[string]interface{}{
				testURLKey: testHookURL,
			},
			Events: []string{"*"},
		},
	}

	assert.Equal(t, hooks[1], getHook(hooks, testHookURL, []string{"*"}))
	assert.Nil(t, getHook(hooks, testHookURL, []string{"does_not_exist"}))
}
