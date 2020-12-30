/*
Copyright 2020 BlackRock, Inc.

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

package policy

import (
	"context"

	"github.com/pkg/errors"
)

// StatusPolicy implements the policy for a HTTP trigger
type StatusPolicy struct {
	// Status represents the status of a generic operation
	Status int
	// Statuses refers to list of response status allowed
	Statuses []int
}

// NewStatusPolicy returns a new HTTP trigger policy
func NewStatusPolicy(status int, statuses []int) *StatusPolicy {
	return &StatusPolicy{
		Status:   status,
		Statuses: statuses,
	}
}

func (hp *StatusPolicy) ApplyPolicy(ctx context.Context) error {
	for _, status := range hp.Statuses {
		if hp.Status == status {
			return nil
		}
	}
	return errors.Errorf("policy application resulted in failure. http response status %d is not allowed", hp.Status)
}
