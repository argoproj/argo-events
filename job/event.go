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

package job

import (
	"time"
)

// Event interface enables access to events from any signal source
// Events are sent from the signal Goroutines -> sensor executor listeners
type Event interface {
	// GetID should be implemented by the instance events
	// the ID should be the nodeID of the Signal
	// used for updating the signal node status of the sensor resource
	GetID() string

	// GetSource of the signal
	// the source can be the name of the Signal or other information relating to event's source
	GetSource() string

	GetContentType() string
	GetBody() []byte
	GetHeader(string) interface{}
	GetTimestamp() time.Time

	SetError(err error)
	GetError() error

	// GetSignal returns a reference to the Signal that caused this event
	GetSignal() Signal
}

// AbstractEvent is the base implementation of an event
type AbstractEvent struct {
	id             string
	emptyByteArray []byte
	emptyHeaders   map[string]interface{}
	err            error
	done           bool
}

// GetContentType of the event
func (ae *AbstractEvent) GetContentType() string {
	return ""
}

// GetBody of the event
func (ae *AbstractEvent) GetBody() []byte {
	return ae.emptyByteArray
}

// GetHeader of the event
func (ae *AbstractEvent) GetHeader(string) interface{} {
	return nil
}

// SetError on the event
func (ae *AbstractEvent) SetError(err error) {
	ae.err = err
}

// GetError of the event
func (ae *AbstractEvent) GetError() error {
	return ae.err
}
