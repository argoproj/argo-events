package github

import (
	"sort"

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

	uniqFirst := unique(first)
	uniqSecond := unique(second)

	sort.Strings(uniqFirst)
	sort.Strings(uniqSecond)

	// all_request
	if contains(uniqFirst, "*") && contains(uniqSecond, "*") {
		return true
	}

	if len(uniqFirst) != len(uniqSecond) {
		return false
	}

	for index := range uniqFirst {
		if uniqFirst[index] != uniqSecond[index] {
			return false
		}
	}

	return true
}

func contains(sortedList []string, str string) bool {
	i := sort.SearchStrings(sortedList, str)
	return i < len(sortedList) && sortedList[i] == str
}

func unique(stringSlice []string) []string {
	if len(stringSlice) == 0 {
		return stringSlice
	}
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
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
