package common

import (
	"encoding/json"
	"time"
)

// Backoff for an operation
type Backoff struct {
	// Duration is the duration in nanoseconds
	Duration time.Duration `json:"duration" protobuf:"varint,1,opt,name=duration,casttype=time.Duration"`
	// Duration is multiplied by factor each iteration
	Factor json.Number `json:"factor" protobuf:"bytes,5,opt,name=factor,casttype=encoding/json.Number"`
	// The amount of jitter applied each iteration
	Jitter *json.Number `json:"jitter,omitempty" protobuf:"bytes,6,opt,name=jitter,casttype=encoding/json.Number"`
	// Exit with error after this many steps
	Steps int32 `json:"steps,omitempty" protobuf:"varint,4,opt,name=steps"`
}

func (b Backoff) GetSteps() int {
	return int(b.Steps)
}
