package sdk

import "errors"

const (
	// ErrMissingAttribute occurs when certain required attributes are missing from a signal specification
	ErrMissingAttribute = "signal missing required attribute %s"
)

var (
	ErrMissingRequiredAttribute = errors.New("signal missing required attribute %s")
)
