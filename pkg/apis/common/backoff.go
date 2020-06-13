package common

import "time"

// Backoff for an operation
type Backoff struct {
	// Duration is the duration in nanoseconds
	Duration time.Duration `json:"duration" protobuf:"bytes,1,name=duration"`
	// Duration is multiplied by factor each iteration
	Factor float64 `json:"factor" protobuf:"bytes,2,name=factor"`
	// The amount of jitter applied each iteration
	Jitter float64 `json:"jitter,omitempty" protobuf:"bytes,3,opt,name=jitter"`
	// Exit with error after this many steps
	Steps int32 `json:"steps,omitempty" protobuf:"bytes,4,opt,name=steps"`
}

func (b Backoff) GetSteps() int {
	return int(b.Steps)
}
