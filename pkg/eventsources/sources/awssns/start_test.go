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

package awssns

import (
	"testing"
)

func Test_httpNotification_verifySigningCertUrl(t *testing.T) {
	type fields struct {
		SigningCertURL string
	}
	tests := map[string]struct {
		fields  fields
		wantErr bool
	}{
		"valid":             {fields{"https://sns.us-west-2.amazonaws.com/SimpleNotificationService-123.pem"}, false},
		"without https":     {fields{"http://sns.us-west-2.amazonaws.com/SimpleNotificationService-123.pem"}, true},
		"invalid hostname":  {fields{"https://sns.us-west-2.amazonaws-malicious.com/SimpleNotificationService-123.pem"}, true},
		"invalid subdomain": {fields{"https://other.us-west-2.amazonaws.com/SimpleNotificationService-123.pem"}, true},
	}
	for name, tt := range tests {
		name, tt := name, tt
		t.Run(name, func(t *testing.T) {
			m := &httpNotification{
				SigningCertURL: tt.fields.SigningCertURL,
			}
			if err := m.verifySigningCertUrl(); (err != nil) != tt.wantErr {
				t.Errorf("httpNotification.verifySigningCertUrl() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
