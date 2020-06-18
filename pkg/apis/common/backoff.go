package common

import (
	"time"
)

// Backoff for an operation
type Backoff struct {
	// Duration is the duration in nanoseconds
	Duration time.Duration `json:"duration" protobuf:"varint,1,opt,name=duration,casttype=time.Duration"`
	// Duration is multiplied by factor each iteration
	Factor Amount `json:"factor" protobuf:"bytes,2,opt,name=factor"`
	// The amount of jitter applied each iteration
	Jitter *Amount `json:"jitter,omitempty" protobuf:"bytes,3,opt,name=jitter"`
	// Exit with error after this many steps
	Steps int32 `json:"steps,omitempty" protobuf:"varint,4,opt,name=steps"`
}

func (b Backoff) GetSteps() int {
	return int(b.Steps)
}
