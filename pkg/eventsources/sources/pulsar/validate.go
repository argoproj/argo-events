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
package pulsar

import (
	"context"
	"fmt"
)

func (el *EventListener) ValidateEventSource(ctx context.Context) error {
	es := el.PulsarEventSource
	if es.Topics == nil {
		return fmt.Errorf("topics can't be empty list")
	}
	if es.URL == "" {
		return fmt.Errorf("url must be provided")
	}
	if es.Type != "exclusive" && es.Type != "shared" {
		return fmt.Errorf("subscription type must be either exclusive or shared")
	}
	return nil
}
