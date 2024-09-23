package fsevent

import (
	"bytes"
	"fmt"
	"strings"
)

// Event represents a single file system notification.
type Event struct {
	// Relative path to the file or directory.
	Name string `json:"name"`
	// File operation that triggered the event.
	Op Op `json:"op"`
	// User metadata
	Metadata map[string]string `json:"metadata"`
}

// Op describes a set of file operations.
type Op uint32

// These are the generalized file operations that can trigger a notification.
const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Chmod
)

func (op Op) String() string {
	// Use a buffer for efficient string concatenation
	var buffer bytes.Buffer

	if op&Create == Create {
		buffer.WriteString("|CREATE")
	}
	if op&Remove == Remove {
		buffer.WriteString("|REMOVE")
	}
	if op&Write == Write {
		buffer.WriteString("|WRITE")
	}
	if op&Rename == Rename {
		buffer.WriteString("|RENAME")
	}
	if op&Chmod == Chmod {
		buffer.WriteString("|CHMOD")
	}
	if buffer.Len() == 0 {
		return ""
	}
	return buffer.String()[1:] // Strip leading pipe
}

// NewOp converts a given string to Op
func NewOp(s string) Op {
	var op Op
	for _, ss := range strings.Split(s, "|") {
		switch ss {
		case "CREATE":
			op |= Create
		case "REMOVE":
			op |= Remove
		case "WRITE":
			op |= Write
		case "RENAME":
			op |= Rename
		case "CHMOD":
			op |= Chmod
		}
	}
	return op
}

// String returns a string representation of the event in the form
// "file: REMOVE|WRITE|..."
func (e Event) String() string {
	return fmt.Sprintf("%q: %s", e.Name, e.Op.String())
}
