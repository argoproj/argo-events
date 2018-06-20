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

package calendar

import (
	"time"

	"github.com/argoproj/argo-events/job"
)

type event struct {
	job.AbstractEvent
	calendar  *calendar
	timestamp time.Time
}

func (e *event) GetID() string {
	return e.calendar.AbstractSignal.GetID()
}

func (e *event) GetSource() string {
	return e.calendar.getSource()
}

func (e *event) GetTimestamp() time.Time {
	return e.timestamp
}

func (e *event) GetSignal() job.Signal {
	return e.calendar
}
