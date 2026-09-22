package github

import (
	"context"

	gh "github.com/google/go-github/v50/github"

	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

// hooksPerPage is the page size used when listing existing webhooks. It is the
// maximum page size accepted by the GitHub API.
const hooksPerPage = 100

// listAllHooks collects every page of a paginated GitHub hook listing.
//
// The GitHub API returns only the first 30 items when no pagination options are
// given, so a repository or organization with more webhooks than that can have
// its existing hook go unnoticed, which results in a new duplicate hook being
// created on every event source restart.
func listAllHooks(ctx context.Context, list func(context.Context, *gh.ListOptions) ([]*gh.Hook, *gh.Response, error)) ([]*gh.Hook, error) {
	opts := &gh.ListOptions{PerPage: hooksPerPage, Page: 1}

	var all []*gh.Hook
	for {
		hooks, resp, err := list(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, hooks...)

		// resp.NextPage is 0 on the last page. The additional comparison guards
		// against a server response that would otherwise loop forever.
		if resp == nil || resp.NextPage <= opts.Page {
			return all, nil
		}
		opts.Page = resp.NextPage
	}
}

// listOrganizationHooks returns all webhooks configured on the given organization.
func listOrganizationHooks(ctx context.Context, client *gh.Client, org string) ([]*gh.Hook, error) {
	return listAllHooks(ctx, func(ctx context.Context, opts *gh.ListOptions) ([]*gh.Hook, *gh.Response, error) {
		return client.Organizations.ListHooks(ctx, org, opts)
	})
}

// listRepositoryHooks returns all webhooks configured on the given repository.
func listRepositoryHooks(ctx context.Context, client *gh.Client, owner, repo string) ([]*gh.Hook, error) {
	return listAllHooks(ctx, func(ctx context.Context, opts *gh.ListOptions) ([]*gh.Hook, *gh.Response, error) {
		return client.Repositories.ListHooks(ctx, owner, repo, opts)
	})
}

// compareHook returns true if the hook matches the url and event.
func compareHook(hook *gh.Hook, url string, events []string) bool {
	if hook == nil {
		return false
	}

	if hook.Config["url"] != url {
		return false
	}

	// Webhook events are equal if both old events slice and new events slice
	// contain the same events, or if both have "*" event.
	return sharedutil.ElementsMatch(hook.Events, events) ||
		(sharedutil.SliceContains(hook.Events, "*") && sharedutil.SliceContains(events, "*"))
}

// getHook returns the hook that matches the url and event, or nil if not found.
func getHook(hooks []*gh.Hook, url string, event []string) *gh.Hook {
	for _, hook := range hooks {
		if compareHook(hook, url, event) {
			return hook
		}
	}

	return nil
}
