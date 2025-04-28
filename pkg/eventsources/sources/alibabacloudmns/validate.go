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

package alibabacloudmns

import (
	"context"
	"fmt"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

// ValidateEventSource validates sqs event source
func (listener *EventListener) ValidateEventSource(ctx context.Context) error {
	return validate(&listener.MNSEventSource)
}

func validate(eventSource *aev1.MNSEventSource) error {
	if eventSource == nil {
		return aev1.ErrNilEventSource
	}
	if eventSource.Queue == "" {
		return fmt.Errorf("must specify queue name")
	}
	if eventSource.Endpoint == "" {
		return fmt.Errorf("must specify a endpoint")
	}
	return nil
}
