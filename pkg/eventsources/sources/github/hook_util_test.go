package github

import (
	"context"
	"errors"
	"testing"

	gh "github.com/google/go-github/v50/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestListAllHooksSinglePage(t *testing.T) {
	calls := 0
	hooks, err := listAllHooks(context.Background(), func(ctx context.Context, opts *gh.ListOptions) ([]*gh.Hook, *gh.Response, error) {
		calls++
		assert.Equal(t, hooksPerPage, opts.PerPage)
		assert.Equal(t, 1, opts.Page)
		return []*gh.Hook{hookWithURL("https://example0.com/")}, &gh.Response{NextPage: 0}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Len(t, hooks, 1)
}

func TestListAllHooksFollowsPagination(t *testing.T) {
	pages := [][]*gh.Hook{
		{hookWithURL("https://example0.com/")},
		{hookWithURL("https://example1.com/")},
		{hookWithURL("https://example2.com/")},
	}

	hooks, err := listAllHooks(context.Background(), func(ctx context.Context, opts *gh.ListOptions) ([]*gh.Hook, *gh.Response, error) {
		page := pages[opts.Page-1]
		nextPage := opts.Page + 1
		if opts.Page == len(pages) {
			nextPage = 0
		}
		return page, &gh.Response{NextPage: nextPage}, nil
	})

	require.NoError(t, err)
	require.Len(t, hooks, 3)
	// A hook that is only present on a later page must still be found.
	assert.Equal(t, hooks[2], getHook(hooks, "https://example2.com/", []string{"*"}))
}

func TestListAllHooksError(t *testing.T) {
	hooks, err := listAllHooks(context.Background(), func(ctx context.Context, opts *gh.ListOptions) ([]*gh.Hook, *gh.Response, error) {
		return nil, nil, errors.New("boom")
	})

	require.Error(t, err)
	assert.Nil(t, hooks)
}

func TestListAllHooksStopsOnNonAdvancingPage(t *testing.T) {
	calls := 0
	hooks, err := listAllHooks(context.Background(), func(ctx context.Context, opts *gh.ListOptions) ([]*gh.Hook, *gh.Response, error) {
		calls++
		// A server that keeps pointing at the current page must not loop forever.
		return []*gh.Hook{hookWithURL("https://example0.com/")}, &gh.Response{NextPage: opts.Page}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Len(t, hooks, 1)
}

func hookWithURL(url string) *gh.Hook {
	return &gh.Hook{
		Config: map[string]interface{}{"url": url},
		Events: []string{"*"},
	}
}
