package expr

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpand(t *testing.T) {
	for i := 0; i < 1; i++ { // loop 100 times, because map ordering is not determisitic
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			before := map[string]interface{}{
				"a.b":   1,
				"a.c.d": 2,
				"a":     3, // should be deleted
				"ab":    4,
				"abb":   5, // should be kept
			}
			after := Expand(before)
			assert.Len(t, before, 5, "original map unchanged")
			assert.Equal(t, map[string]interface{}{
				"a": map[string]interface{}{
					"b": 1,
					"c": map[string]interface{}{
						"d": 2,
					},
				},
				"ab":  4,
				"abb": 5,
			}, after)
		})
	}
}

func TestEvalBool(t *testing.T) {
	env := map[string]interface{}{
		"id":         1,
		"first_name": "John",
		"last_name":  "Doe",
		"email":      "johndoe@intuit.com",
		"gender":     "Male",
		"dept":       "devp",
		"uuid":       "test-case-hyphen",
	}

	pass, err := EvalBool("(id == 1) && (last_name == 'Doe')", env)
	assert.NoError(t, err)
	assert.True(t, pass)

	pass, err = EvalBool("(id == 2) || (gender == 'Female')", env)
	assert.NoError(t, err)
	assert.False(t, pass)

	pass, err = EvalBool("invalidexpression", env)
	assert.Error(t, err)
	assert.False(t, pass)

	// expr with '-' evaluate the same as others
	pass, err = EvalBool("(uuid == 'test-case-hyphen')", env)
	assert.NoError(t, err)
	assert.True(t, pass)
}

func TestRemoveConflictingKeys(t *testing.T) {
	testCases := []struct {
		name   string
		input  map[string]interface{}
		output map[string]interface{}
	}{
		{
			name: "remove conflicting keys",
			input: map[string]interface{}{
				"a.b": 1,
				"a":   2,
			},
			output: map[string]interface{}{
				"a.b": 1,
			},
		},
		{
			name: "no conflicts",
			input: map[string]interface{}{
				"a":   1,
				"b":   2,
				"c.d": 3,
			},
			output: map[string]interface{}{
				"a":   1,
				"b":   2,
				"c.d": 3,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := removeConflicts(tc.input)
			if !reflect.DeepEqual(result, tc.output) {
				t.Errorf("expected %v, but got %v", tc.output, result)
			}
		})
	}
}
