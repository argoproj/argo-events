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

	"github.com/pkg/errors"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
)

var (
	defaultFactor   = apicommon.NewAmount("1.0")
	defaultJitter   = apicommon.NewAmount("1")
	defaultDuration = apicommon.FromString("1s")

	DefaultBackoff = apicommon.Backoff{
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
func Convert2WaitBackoff(backoff *apicommon.Backoff) (*wait.Backoff, error) {
	result := wait.Backoff{}

	d := backoff.Duration
	if d == nil {
		d = &defaultDuration
	}
	if d.Type == apicommon.Int64 {
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
		return nil, errors.Wrap(err, "invalid factor")
	}
	result.Factor = f

	jitter := backoff.Jitter
	if jitter == nil {
		jitter = &defaultJitter
	}
	j, err := jitter.Float64()
	if err != nil {
		return nil, errors.Wrap(err, "invalid jitter")
	}
	result.Jitter = j

	if backoff.Steps > 0 {
		result.Steps = backoff.GetSteps()
	} else {
		result.Steps = int(DefaultBackoff.Steps)
	}
	return &result, nil
}

func Connect(backoff *apicommon.Backoff, conn func() error) error {
	if backoff == nil {
		backoff = &DefaultBackoff
	}
	b, err := Convert2WaitBackoff(backoff)
	if err != nil {
		return errors.Wrap(err, "invalid backoff configuration")
	}
	if waitErr := wait.ExponentialBackoff(*b, func() (bool, error) {
		if err = conn(); err != nil {
			// return "false, err" will cover waitErr
			return false, nil
		}
		return true, nil
	}); waitErr != nil {
		if err != nil {
			return fmt.Errorf("%v: %v", waitErr, err)
		} else {
			return waitErr
		}
	}
	return nil
}
