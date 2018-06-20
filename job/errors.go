package job

import "errors"

const (
	// ErrMissingAttribute occurs when certain required attributes are missing from a signal specification
	ErrMissingAttribute = "%s signal expecting %s but found no such attribute defined"
)

var (
	ErrMissingRequiredAttribute = errors.New("signal missing required attribute")
)
