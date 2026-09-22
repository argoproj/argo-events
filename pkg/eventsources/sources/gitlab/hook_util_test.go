package gitlab

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
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

func TestListAllHooksSinglePage(t *testing.T) {
	calls := 0
	hooks, err := listAllHooks(func(opts gitlab.ListOptions) ([]*gitlab.ProjectHook, *gitlab.Response, error) {
		calls++
		assert.Equal(t, int64(hooksPerPage), opts.PerPage)
		assert.Equal(t, int64(1), opts.Page)
		return []*gitlab.ProjectHook{{URL: "https://example0.com/"}}, &gitlab.Response{NextPage: 0}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Len(t, hooks, 1)
}

func TestListAllHooksFollowsPagination(t *testing.T) {
	pages := [][]*gitlab.ProjectHook{
		{{URL: "https://example0.com/"}},
		{{URL: "https://example1.com/"}},
		{{URL: "https://example2.com/"}},
	}

	hooks, err := listAllHooks(func(opts gitlab.ListOptions) ([]*gitlab.ProjectHook, *gitlab.Response, error) {
		page := pages[opts.Page-1]
		nextPage := opts.Page + 1
		if int(opts.Page) == len(pages) {
			nextPage = 0
		}
		return page, &gitlab.Response{NextPage: nextPage}, nil
	})

	require.NoError(t, err)
	require.Len(t, hooks, 3)
	// A hook that is only present on a later page must still be found.
	assert.Equal(t, hooks[2], getProjectHook(hooks, "https://example2.com/"))
}

func TestListAllHooksError(t *testing.T) {
	hooks, err := listAllHooks(func(opts gitlab.ListOptions) ([]*gitlab.ProjectHook, *gitlab.Response, error) {
		return nil, nil, errors.New("boom")
	})

	require.Error(t, err)
	assert.Nil(t, hooks)
}

func TestListAllHooksStopsOnNonAdvancingPage(t *testing.T) {
	calls := 0
	hooks, err := listAllHooks(func(opts gitlab.ListOptions) ([]*gitlab.GroupHook, *gitlab.Response, error) {
		calls++
		// A server that keeps pointing at the current page must not loop forever.
		return []*gitlab.GroupHook{{URL: "https://example0.com/"}}, &gitlab.Response{NextPage: opts.Page}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Len(t, hooks, 1)
}
