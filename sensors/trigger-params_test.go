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

package sensors

import (
	"reflect"
	"testing"

	"fmt"

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func Test_applyParams(t *testing.T) {
	defaultValue := "default"
	events := map[string]apicommon.Event{
		"simpleJSON": {
			Context: apicommon.EventContext{
				ContentType: MediaTypeJSON,
			},
			Payload: []byte(`{"name":{"first":"matt","last":"magaldi"},"age":24}`),
		},
		"nonJSON": {
			Context: apicommon.EventContext{
				ContentType: MediaTypeJSON,
			},
			Payload: []byte(`apiVersion: v1alpha1`),
		},
	}
	type args struct {
		jsonObj []byte
		params  []v1alpha1.ResourceParameter
		events  map[string]apicommon.Event
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "no event and missing default -> error",
			args: args{
				jsonObj: []byte(""),
				params: []v1alpha1.ResourceParameter{
					{
						Src: &v1alpha1.ResourceParameterSource{
							Event: "missing",
						},
					},
				},
				events: events,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no event with default -> success",
			args: args{
				jsonObj: []byte(""),
				params: []v1alpha1.ResourceParameter{
					{
						Src: &v1alpha1.ResourceParameterSource{
							Event: "missing",
							Value: &defaultValue,
						},
						Dest: "x",
					},
				},
				events: events,
			},
			want:    []byte(`{"x":"default"}`),
			wantErr: false,
		},
		{
			name: "no event with default, but missing dest -> error",
			args: args{
				jsonObj: []byte(""),
				params: []v1alpha1.ResourceParameter{
					{
						Src: &v1alpha1.ResourceParameterSource{
							Event: "missing",
							Value: &defaultValue,
						},
					},
				},
				events: events,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "simpleJSON (new field) -> success",
			args: args{
				jsonObj: []byte(``),
				params: []v1alpha1.ResourceParameter{
					{
						Src: &v1alpha1.ResourceParameterSource{
							Event: "simpleJSON",
							Path:  "name.last",
						},
						Dest: "x",
					},
				},
				events: events,
			},
			want:    []byte(`{"x":"magaldi"}`),
			wantErr: false,
		},
		{
			name: "simpleJSON (updated field) -> success",
			args: args{
				jsonObj: []byte(`{"x":"before"}`),
				params: []v1alpha1.ResourceParameter{
					{
						Src: &v1alpha1.ResourceParameterSource{
							Event: "simpleJSON",
							Path:  "name.last",
						},
						Dest: "x",
					},
				},
				events: events,
			},
			want:    []byte(`{"x":"magaldi"}`),
			wantErr: false,
		},
		{
			name: "non JSON, no default -> pass payload bytes without converting",
			args: args{
				jsonObj: []byte(``),
				params: []v1alpha1.ResourceParameter{
					{
						Src: &v1alpha1.ResourceParameterSource{
							Event: "nonJSON",
						},
						Dest: "x",
					},
				},
				events: events,
			},
			want:    []byte(fmt.Sprintf(`{"x":"%s"}`, string(events["nonJSON"].Payload))),
			wantErr: false,
		},
		{
			name: "non JSON, with path -> error",
			args: args{
				jsonObj: []byte(``),
				params: []v1alpha1.ResourceParameter{
					{
						Src: &v1alpha1.ResourceParameterSource{
							Event: "nonJSON",
							Path:  "test",
						},
						Dest: "x",
					},
				},
				events: events,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyParams(tt.args.jsonObj, tt.args.params, tt.args.events)
			if (err != nil) != tt.wantErr {
				t.Errorf("applyParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("applyParams() = %v, want %v", got, tt.want)
			}
		})
	}
}
