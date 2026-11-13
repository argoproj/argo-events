/*
Copyright 2018 The Argoproj Authors.

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

package webhook

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestValidateWebhook(t *testing.T) {
	convey.Convey("Given a webhook, validate it", t, func() {
		convey.So(ValidateWebhookContext(Hook), convey.ShouldBeNil)
	})
}

func TestNewWebhookHelper(t *testing.T) {
	convey.Convey("Make sure webhook helper is not empty", t, func() {
		controller := NewController()
		convey.So(controller, convey.ShouldNotBeNil)
	})
}

func TestParseBearerAuthorization(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		wantToken string
		wantOK    bool
	}{
		{name: "canonical scheme", header: "Bearer token", wantToken: "token", wantOK: true},
		{name: "lowercase scheme", header: "bearer token", wantToken: "token", wantOK: true},
		{name: "mixed case scheme", header: "bEaReR  token", wantToken: "token", wantOK: true},
		{name: "wrong scheme", header: "Basic token"},
		{name: "missing token", header: "Bearer "},
		{name: "token contains whitespace", header: "Bearer token extra"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotToken, gotOK := parseBearerAuthorization(test.header)
			if gotToken != test.wantToken || gotOK != test.wantOK {
				t.Fatalf("parseBearerAuthorization(%q) = (%q, %t), want (%q, %t)", test.header, gotToken, gotOK, test.wantToken, test.wantOK)
			}
		})
	}
}
