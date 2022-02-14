package expr

import (
	"fmt"
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

}
