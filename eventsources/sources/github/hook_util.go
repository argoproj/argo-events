package github

import (
	gh "github.com/google/go-github/v31/github"

	"github.com/argoproj/argo-events/common"
)

// compareHook returns true if the hook matches the url and event.
func compareHook(hook *gh.Hook, url string, event []string) bool {
	if hook == nil {
		return false
	}

	if hook.Config["url"] != url {
		return false
	}

	return common.AreSlicesEqual(hook.Events, event)
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
