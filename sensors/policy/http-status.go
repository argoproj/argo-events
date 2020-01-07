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
	"net/http"

	"github.com/pkg/errors"
)

// HTTPTriggerPolicy implements the policy for a HTTP trigger
type HTTPTriggerPolicy struct {
	// Response of the http request upon which the policy must be applied
	Response *http.Response
	// Statuses refers to list of response status allowed
	Statuses []string
}

// NewHTTPTriggerPolicy returns a new HTTP trigger policy
func NewHTTPTriggerPolicy(response *http.Response, statuses []string) *HTTPTriggerPolicy {
	return &HTTPTriggerPolicy{
		Response: response,
		Statuses: statuses,
	}
}

func (hp *HTTPTriggerPolicy) ApplyPolicy() error {
	for _, status := range hp.Statuses {
		if hp.Response.Status == status {
			return nil
		}
	}
	return errors.Errorf("policy application resulted in failure. http response status %s is not allowed", hp.Response.Status)
}
