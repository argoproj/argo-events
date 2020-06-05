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
	"strconv"
	"time"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Backoff defines an operational backoff
type Backoff struct {
	Duration time.Duration `json:"duration"` // the base duration
	Factor   string        `json:"factor"`   // Duration is multiplied by factor each iteration
	Jitter   string        `json:"jitter"`   // The amount of jitter applied each iteration
	Steps    int           `json:"steps"`    // Exit with error after this many steps
}

func (b Backoff) GetFactor() float64 {
	value, err := strconv.ParseFloat(b.Factor, 10)
	if err != nil {
		return 0
	}
	return value
}

func (b Backoff) GetJitter() float64 {
	value, err := strconv.ParseFloat(b.Jitter, 10)
	if err != nil {
		return 0
	}
	return value
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

// GetConnectionBackoff returns a connection backoff option
func GetConnectionBackoff(backoff *Backoff) *wait.Backoff {
	result := wait.Backoff{
		Duration: DefaultRetry.Duration,
		Factor:   DefaultRetry.Factor,
		Jitter:   DefaultRetry.Jitter,
		Steps:    DefaultRetry.Steps,
	}
	if backoff == nil {
		return &result
	}
	if &backoff.Duration != nil {
		result.Duration = backoff.Duration
	}
	if backoff.Factor != "" {
		result.Factor = backoff.GetFactor()
	}
	if backoff.Jitter != "" {
		result.Jitter = backoff.GetJitter()
	}
	if backoff.Steps != 0 {
		result.Steps = backoff.Steps
	}
	return &result
}
