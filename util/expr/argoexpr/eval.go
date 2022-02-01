package argoexpr

import (
	"fmt"

	"sort"
	"strings"

	"encoding/json"

	"github.com/antonmedv/expr"
	"github.com/doublerebel/bellows"

	sprig "github.com/Masterminds/sprig/v3"
	exprpkg "github.com/argoproj/pkg/expr"
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
	for k, v := range exprpkg.GetExprEnvFunctionMap() {
		env[k] = v
	}
	env["toJson"] = toJson
	env["sprig"] = sprigFuncMap
	return env
}

func toJson(v interface{}) string {
	output, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(output)
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
