/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

// Backoff for an operation
type Backoff struct {
	// The initial duration in nanoseconds or strings like "1s", "3m"
	// +optional
	Duration *Int64OrString `json:"duration,omitempty" protobuf:"bytes,1,opt,name=duration"`
	// Duration is multiplied by factor each iteration
	// +optional
	Factor *Amount `json:"factor,omitempty" protobuf:"bytes,2,opt,name=factor"`
	// The amount of jitter applied each iteration
	// +optional
	Jitter *Amount `json:"jitter,omitempty" protobuf:"bytes,3,opt,name=jitter"`
	// Exit with error after this many steps
	// +optional
	Steps int32 `json:"steps,omitempty" protobuf:"varint,4,opt,name=steps"`
}

func (b Backoff) GetSteps() int {
	return int(b.Steps)
}
