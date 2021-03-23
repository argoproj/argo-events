package github

import (
	gh "github.com/google/go-github/v31/github"
)

// sliceEqual returns true if the two provided string slices are equal.
func sliceEqual(first []string, second []string) bool {
	if len(first) == 0 && len(second) == 0 {
		return true
	}
	if len(first) == 0 || len(second) == 0 {
		return false
	}

	tmp := make(map[string]int)
	for _, i := range first {
		tmp[i] = 1
	}

	for _, i := range second {
		v, ok := tmp[i]
		if !ok || v == -1 {
			tmp[i] = -1
		} else {
			tmp[i] = 2
		}
	}

	if v, ok := tmp["*"]; ok {
		// If both slices contain "*", return true directly
		return v == 2
	}

	for _, v := range tmp {
		// -1: only exists in secod
		// 1: only exists in first
		// 2: eists in both
		if v < 2 {
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
