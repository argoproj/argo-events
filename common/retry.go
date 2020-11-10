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

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
)

// DefaultRetry is a default retry backoff settings when retrying API calls
var DefaultRetry = wait.Backoff{
	Steps:    5,
	Duration: 1 * time.Second,
	Factor:   1.0,
	Jitter:   1,
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
func GetConnectionBackoff(backoff *apicommon.Backoff) *wait.Backoff {
	result := wait.Backoff{
		Duration: DefaultRetry.Duration,
		Factor:   DefaultRetry.Factor,
		Jitter:   DefaultRetry.Jitter,
		Steps:    DefaultRetry.Steps,
	}
	if backoff == nil {
		return &result
	}
	result.Duration = backoff.Duration
	result.Factor, _ = backoff.Factor.Float64()
	if backoff.Jitter != nil {
		result.Jitter, _ = backoff.Jitter.Float64()
	}
	if backoff.Steps != 0 {
		result.Steps = backoff.GetSteps()
	}
	return &result
}

// Connect is a general connection helper
func Connect(backoff *wait.Backoff, conn func() error) error {
	if backoff == nil {
		backoff = &DefaultRetry
	}
	err := wait.ExponentialBackoff(*backoff, func() (bool, error) {
		if err := conn(); err != nil {
			return false, nil
		}
		return true, nil
	})
	return err
}
