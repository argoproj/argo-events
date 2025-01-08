/*
Copyright 2025 The Argoproj Authors.

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

package awssns

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/argoproj/argo-events/pkg/eventsources/common/webhook"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

func FuzzAWSSNSsource(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		fRoute := webhook.GetFakeRoute()
		router := &Router{
			Route:       fRoute,
			eventSource: &v1alpha1.SNSEventSource{},
		}
		router.Route.Active = true
		writer := &webhook.FakeHttpWriter{}
		r := &http.Request{
			Body: io.NopCloser(bytes.NewReader(data)),
		}
		r.Header = make(map[string][]string)
		r.Header.Set("Content-Type", "application/json")
		router.HandleRoute(writer, r)
	})
}
