package gitlab

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// hooksPerPage is the page size used when listing existing webhooks. It is the
// maximum page size accepted by the GitLab API.
const hooksPerPage = 100

// listAllHooks collects every page of a paginated GitLab hook listing.
//
// The GitLab API returns only the first 20 items when no pagination options are
// given, so a project or group with more webhooks than that can have its
// existing hook go unnoticed, which results in a new duplicate hook being
// created on every event source restart.
func listAllHooks[T any](list func(gitlab.ListOptions) ([]*T, *gitlab.Response, error)) ([]*T, error) {
	opts := gitlab.ListOptions{PerPage: hooksPerPage, Page: 1}

	var all []*T
	for {
		hooks, resp, err := list(opts)
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

// listProjectHooks returns all webhooks configured on the given project.
func listProjectHooks(client *gitlab.Client, project string) ([]*gitlab.ProjectHook, error) {
	return listAllHooks(func(opts gitlab.ListOptions) ([]*gitlab.ProjectHook, *gitlab.Response, error) {
		return client.Projects.ListProjectHooks(project, &gitlab.ListProjectHooksOptions{ListOptions: opts})
	})
}

// listGroupHooks returns all webhooks configured on the given group.
func listGroupHooks(client *gitlab.Client, group string) ([]*gitlab.GroupHook, error) {
	return listAllHooks(func(opts gitlab.ListOptions) ([]*gitlab.GroupHook, *gitlab.Response, error) {
		return client.Groups.ListGroupHooks(group, &gitlab.ListGroupHooksOptions{ListOptions: opts})
	})
}

func getProjectHook(hooks []*gitlab.ProjectHook, url string) *gitlab.ProjectHook {
	for _, h := range hooks {
		if h.URL != url {
			continue
		}
		return h
	}
	return nil
}

func getGroupHook(hooks []*gitlab.GroupHook, url string) *gitlab.GroupHook {
	for _, h := range hooks {
		if h.URL != url {
			continue
		}
		return h
	}
	return nil
}
