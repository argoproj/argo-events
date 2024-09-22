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
	"fmt"
	"time"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

var (
	defaultFactor   = aev1.NewAmount("1.0")
	defaultJitter   = aev1.NewAmount("1")
	defaultDuration = aev1.FromString("1s")

	DefaultBackoff = aev1.Backoff{
		Steps:    5,
		Duration: &defaultDuration,
		Factor:   &defaultFactor,
		Jitter:   &defaultJitter,
	}
)

// IsRetryableKubeAPIError returns if the error is a retryable kubernetes error
func IsRetryableKubeAPIError(err error) bool {
	// get original error if it was wrapped
	if apierr.IsNotFound(err) || apierr.IsForbidden(err) || apierr.IsInvalid(err) || apierr.IsMethodNotSupported(err) {
		return false
	}
	return true
}

// Convert2WaitBackoff converts to a wait backoff option
func Convert2WaitBackoff(backoff *aev1.Backoff) (*wait.Backoff, error) {
	result := wait.Backoff{}

	d := backoff.Duration
	if d == nil {
		d = &defaultDuration
	}
	if d.Type == aev1.Int64 {
		result.Duration = time.Duration(d.Int64Value())
	} else {
		parsedDuration, err := time.ParseDuration(d.StrVal)
		if err != nil {
			return nil, err
		}
		result.Duration = parsedDuration
	}

	factor := backoff.Factor
	if factor == nil {
		factor = &defaultFactor
	}
	f, err := factor.Float64()
	if err != nil {
		return nil, fmt.Errorf("invalid factor, %w", err)
	}
	result.Factor = f

	jitter := backoff.Jitter
	if jitter == nil {
		jitter = &defaultJitter
	}
	j, err := jitter.Float64()
	if err != nil {
		return nil, fmt.Errorf("invalid jitter, %w", err)
	}
	result.Jitter = j

	if backoff.Steps > 0 {
		result.Steps = backoff.GetSteps()
	} else {
		result.Steps = int(DefaultBackoff.Steps)
	}
	return &result, nil
}

func DoWithRetry(backoff *aev1.Backoff, f func() error) error {
	if backoff == nil {
		backoff = &DefaultBackoff
	}
	b, err := Convert2WaitBackoff(backoff)
	if err != nil {
		return fmt.Errorf("invalid backoff configuration, %w", err)
	}
	_ = wait.ExponentialBackoff(*b, func() (bool, error) {
		if err = f(); err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed after retries: %w", err)
	}
	return nil
}
