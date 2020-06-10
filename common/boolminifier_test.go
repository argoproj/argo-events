package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimplifyBoolExpression(t *testing.T) {
	tests := []struct {
		expression string
		expect     string
	}{
		{
			expression: "a || b",
			expect:     "b || a",
		},
		{
			expression: "a && b",
			expect:     "a && b",
		},
		{
			expression: "(a || b || c || d || e) && (c && a)",
			expect:     "a && c",
		},
		{
			expression: "(a || b) && c",
			expect:     "(b && c) || (a && c)",
		},
	}

	for _, test := range tests {
		expr, err := NewBoolExpression(test.expression)
		assert.NoError(t, err)
		assert.Equal(t, test.expect, expr.GetExpression())
	}
}
