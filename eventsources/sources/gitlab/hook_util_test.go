package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
)

func TestGetGroupHook(t *testing.T) {
	hooks := []*gitlab.GroupHook{
		{
			URL: "https://example0.com/",
		},
		{
			URL: "https://example1.com/",
		},
	}

	assert.Equal(t, hooks[1], getGroupHook(hooks, "https://example1.com/"))
	assert.Nil(t, getGroupHook(hooks, "https://example.com/"))
}

func TestGetProjectHook(t *testing.T) {
	hooks := []*gitlab.ProjectHook{
		{
			URL: "https://example0.com/",
		},
		{
			URL: "https://example1.com/",
		},
	}

	assert.Equal(t, hooks[1], getProjectHook(hooks, "https://example1.com/"))
	assert.Nil(t, getProjectHook(hooks, "https://example.com/"))
}
