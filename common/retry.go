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

import (
	"time"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Backoff defines an operational backoff
type Backoff struct {
	Duration time.Duration `json:"duration"` // the base duration
	Factor   float64       `json:"factor"`   // Duration is multiplied by factor each iteration
	Jitter   float64       `json:"jitter"`   // The amount of jitter applied each iteration
	Steps    int           `json:"steps"`    // Exit with error after this many steps
}

// DefaultRetry is a default retry backoff settings when retrying API calls
var DefaultRetry = wait.Backoff{
	Steps:    5,
	Duration: 10 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

// IsRetryableKubeAPIError returns if the error is a retryable kubernetes error
func IsRetryableKubeAPIError(err error) bool {
	// get original error if it was wrapped
	if apierr.IsNotFound(err) || apierr.IsForbidden(err) || apierr.IsInvalid(err) || apierr.IsMethodNotSupported(err) {
		return false
	}
	return true
}
