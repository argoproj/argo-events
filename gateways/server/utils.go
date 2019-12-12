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

package server

import (
	"github.com/argoproj/argo-events/common"
	"k8s.io/apimachinery/pkg/util/wait"
)

// General connection helper
func Connect(backoff *wait.Backoff, conn func() error) error {
	if backoff == nil {
		backoff = &common.DefaultRetry
	}
	err := wait.ExponentialBackoff(*backoff, func() (bool, error) {
		if err := conn(); err != nil {
			return false, nil
		}
		return true, nil
	})
	return err
}
