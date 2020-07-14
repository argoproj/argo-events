package github

import (
	gh "github.com/google/go-github/v31/github"
)

// sliceEqual returns true if the two provided string slices are equal.
func sliceEqual(first []string, second []string) bool {
	if len(first) != len(second) {
		return false
	}

	if first == nil && second == nil {
		return true
	}

	if first == nil || second == nil {
		return false
	}

	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}

	return true
}

// compareHook returns true if the hook matches the url and event.
func compareHook(hook *gh.Hook, url string, event []string) bool {
	if hook == nil {
		return false
	}

	if hook.Config["url"] != url {
		return false
	}

	return sliceEqual(hook.Events, event)
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
