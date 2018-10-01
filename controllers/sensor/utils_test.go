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
package sensor

import (
	"reflect"
	"testing"

	v1alpha "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

// this test is meant to cover the missing cases for those not covered in signal-filter_test.go and trigger-params_test.go
func Test_renderEventDataAsJSON(t *testing.T) {
	type args struct {
		e *v1alpha.Event
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name:    "nil event",
			args:    args{e: nil},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing content type",
			args:    args{e: &v1alpha.Event{}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "valid yaml content",
			args: args{e: &v1alpha.Event{
				Context: v1alpha.EventContext{
					ContentType: MediaTypeYAML,
				},
				Payload: []byte(`apiVersion: v1alpha1`),
			}},
			want:    []byte(`{"apiVersion":"v1alpha1"}`),
			wantErr: false,
		},
		{
			name: "json content marked as yaml",
			args: args{e: &v1alpha.Event{
				Context: v1alpha.EventContext{
					ContentType: MediaTypeYAML,
				},
				Payload: []byte(`{"apiVersion":5}`),
			}},
			want:    []byte(`{"apiVersion":5}`),
			wantErr: false,
		},
		{
			name: "invalid json content",
			args: args{e: &v1alpha.Event{
				Context: v1alpha.EventContext{
					ContentType: MediaTypeJSON,
				},
				Payload: []byte(`{5:"numberkey"}`),
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid yaml content",
			args: args{e: &v1alpha.Event{
				Context: v1alpha.EventContext{
					ContentType: MediaTypeYAML,
				},
				Payload: []byte(`%\x786`),
			}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderEventDataAsJSON(tt.args.e)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderEventDataAsJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("renderEventDataAsJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}
