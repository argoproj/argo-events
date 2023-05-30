package fsevent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpString(t *testing.T) {
	tests := []struct {
		op       Op
		expected string
	}{
		{Create, "CREATE"},
		{Remove, "REMOVE"},
		{Write, "WRITE"},
		{Rename, "RENAME"},
		{Chmod, "CHMOD"},
		{Create | Write, "CREATE|WRITE"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.op.String(), "Op.String() for op %d", tt.op)
	}
}

func TestNewOp(t *testing.T) {
	tests := []struct {
		input    string
		expected Op
	}{
		{"CREATE", Create},
		{"REMOVE", Remove},
		{"WRITE", Write},
		{"RENAME", Rename},
		{"CHMOD", Chmod},
		{"CREATE|WRITE", Create | Write},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, NewOp(tt.input), "NewOp(%q)", tt.input)
	}
}

func TestEventString(t *testing.T) {
	tests := []struct {
		event    Event
		expected string
	}{
		{Event{Name: "file1", Op: Create}, `"file1": CREATE`},
		{Event{Name: "file2", Op: Remove | Write}, `"file2": REMOVE|WRITE`},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.event.String(), "Event.String() for event %#v", tt.event)
	}
}
