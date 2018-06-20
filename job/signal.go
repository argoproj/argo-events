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
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// Signal is the interface for signaling
type Signal interface {
	// GetID returns the nodeID for this signal
	GetID() string

	// Start the signal processing. Events are sent on the Event channel.
	Start(chan Event) error

	// Stop the signal from sending any more events. Terminates all goroutines internal to the signal processing
	Stop() error

	// CheckConstraints for the signal
	// Returns a true if the signal passes constraints
	CheckConstraints(time.Time) bool
}

// AbstractSignal is the base implementation of a signal
type AbstractSignal struct {
	v1alpha1.Signal
	id      string
	Log     *zap.Logger
	Session *ExecutorSession //todo: make private
}

// GetID of the signal
func (as *AbstractSignal) GetID() string {
	return as.id
}

// ErrFailedTimeConstraint is the error for events that do not pass the CheckConstraints method
var ErrFailedTimeConstraint = errors.New("failed time constraint check")

// CheckConstraints for the signal
// returns false if the constraint check fails
func (as *AbstractSignal) CheckConstraints(snapshot time.Time) bool {
	// should figure out why the constraints start and stop time are showing up with the same weird values that are not zero valued
	tConstraints := as.Constraints.Time
	as.Log.Debug("checking", zap.Time("timestamp", snapshot), zap.Time("start", tConstraints.Start.Time), zap.Time("stop", tConstraints.Stop.Time))
	if tConstraints.Start.IsZero() {
		if tConstraints.Stop.IsZero() {
			// no time constraints set
			return true
		}
		// stop contraint only set
		return tConstraints.Stop.Time.After(snapshot)
	}
	if tConstraints.Stop.IsZero() {
		// start constraint only set
		return tConstraints.Start.Time.Before(snapshot)
	}
	// both start and stop set
	return tConstraints.Start.Time.Before(snapshot) && tConstraints.Stop.Time.After(snapshot)
}
