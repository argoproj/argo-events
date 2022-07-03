package github

import (
	gh "github.com/google/go-github/v31/github"

	"github.com/argoproj/argo-events/common"
)

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
	return common.ElementsMatch(hook.Events, events) ||
		(common.SliceContains(hook.Events, "*") && common.SliceContains(events, "*"))
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
