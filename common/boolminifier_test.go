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
			expression: "a_a || b_b",
			expect:     "b_b || a_a",
		},
		{
			expression: "a-a && b-b",
			expect:     "a-a && b-b",
		},
		{
			expression: "(a || b || c || d || e) && (c && a)",
			expect:     "a && c",
		},
		{
			expression: "(a || b) && c",
			expect:     "(b && c) || (a && c)",
		},
		{
			expression: "((a && b) || (c && d)) && c",
			expect:     "(c && d) || (a && b && c)",
		},
		{
			expression: "((a && b) || (c && d)) || c",
			expect:     "c || (a && b)",
		},
		{
			expression: "a:a && b:b",
			expect:     "a:a && b:b",
		},
	}

	for _, test := range tests {
		expr, err := NewBoolExpression(test.expression)
		assert.NoError(t, err)
		assert.Equal(t, test.expect, expr.GetExpression())
	}
}
