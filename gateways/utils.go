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

package gateways

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// SetValidateReason set the result of event source validation
func SetValidEventSource(v *ValidEventSource, reason string, valid bool) {
	v.Reason = reason
	v.IsValid = valid
}

// InitBackoff initializes backoff
func InitBackoff(backoff *wait.Backoff) {
	if backoff == nil {
		backoff = &wait.Backoff{
			Steps:    1,
			Duration: 1,
		}
	}
	backoff.Duration = backoff.Duration * time.Second
}

// General connection helper
func Connect(backoff *wait.Backoff, conn func() error) error {
	InitBackoff(backoff)
	err := wait.ExponentialBackoff(*backoff, func() (bool, error) {
		if err := conn(); err != nil {
			return false, nil
		}
		return true, nil
	})
	return err
}
