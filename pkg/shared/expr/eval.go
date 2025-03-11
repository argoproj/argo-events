package expr

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/builtin"
	"github.com/doublerebel/bellows"
	"github.com/evilmonkeyinc/jsonpath"
)

func EvalBool(input string, env interface{}) (bool, error) {
	result, err := expr.Eval(input, env)
	if err != nil {
		return false, fmt.Errorf("unable to evaluate expression '%s': %s", input, err)
	}
	resultBool, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("unable to cast expression result '%s' to bool", result)
	}
	return resultBool, nil
}

var sprigFuncMap = sprig.GenericFuncMap() // a singleton for better performance

func init() {
	delete(sprigFuncMap, "env")
	delete(sprigFuncMap, "expandenv")
}

func GetFuncMap(m map[string]interface{}) map[string]interface{} {
	env := Expand(m)
	// Alias for the built-in `int` function, for backwards compatibility.
	env["asInt"] = builtin.Int
	// Alias for the built-in `float` function, for backwards compatibility.
	env["asFloat"] = builtin.Float
	env["jsonpath"] = jsonPath
	env["toJson"] = toJson
	env["sprig"] = sprigFuncMap
	return env
}

func toJson(v interface{}) string {
	output, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(output)
}

func jsonPath(jsonStr string, path string) interface{} {
	var jsonMap interface{}
	err := json.Unmarshal([]byte(jsonStr), &jsonMap)
	if err != nil {
		panic(err)
	}
	value, err := jsonpath.Query(path, jsonMap)
	if err != nil {
		panic(err)
	}
	return value
}

func Expand(m map[string]interface{}) map[string]interface{} {
	return bellows.Expand(removeConflicts(m))
}

// It is possible for the map to contain conflicts:
// {"a.b": 1, "a": 2}
// What should the result be? We remove the less-specific key.
// {"a.b": 1, "a": 2} -> {"a.b": 1, "a": 2}
func removeConflicts(m map[string]interface{}) map[string]interface{} {
	var keys []string
	n := map[string]interface{}{}
	for k, v := range m {
		keys = append(keys, k)
		n[k] = v
	}
	sort.Strings(keys)
	for i := 0; i < len(keys)-1; i++ {
		k := keys[i]
		// remove any parent that has a child
		if strings.HasPrefix(keys[i+1], k+".") {
			delete(n, k)
		}
	}
	return n
}
