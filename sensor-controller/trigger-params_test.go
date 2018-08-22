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
package sensor_controller

import (
	"reflect"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	v1alpha "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

func Test_applyParams(t *testing.T) {
	defaultValue := "default"
	events := map[string]v1alpha.Event{
		"simpleJSON": v1alpha.Event{
			Context: v1alpha.EventContext{
				ContentType: MediaTypeJSON,
			},
			Payload: []byte(`{"name":{"first":"matt","last":"magaldi"},"age":24}`),
		},
		"invalidJSON": v1alpha.Event{
			Context: v1alpha.EventContext{
				ContentType: MediaTypeJSON,
			},
			Payload: []byte(`apiVersion: v1alpha1`),
		},
	}
	type args struct {
		jsonObj []byte
		params  []v1alpha1.ResourceParameter
		events  map[string]v1alpha.Event
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
					v1alpha1.ResourceParameter{
						Src: &v1alpha1.ResourceParameterSource{
							Signal: "missing",
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
					v1alpha1.ResourceParameter{
						Src: &v1alpha1.ResourceParameterSource{
							Signal: "missing",
							Value:  &defaultValue,
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
					v1alpha1.ResourceParameter{
						Src: &v1alpha1.ResourceParameterSource{
							Signal: "missing",
							Value:  &defaultValue,
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
					v1alpha1.ResourceParameter{
						Src: &v1alpha1.ResourceParameterSource{
							Signal: "simpleJSON",
							Path:   "name.last",
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
					v1alpha1.ResourceParameter{
						Src: &v1alpha1.ResourceParameterSource{
							Signal: "simpleJSON",
							Path:   "name.last",
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
			name: "invalidJSON, no default -> error",
			args: args{
				jsonObj: []byte(``),
				params: []v1alpha1.ResourceParameter{
					v1alpha1.ResourceParameter{
						Src: &v1alpha1.ResourceParameterSource{
							Signal: "invalidJSON",
						},
						Dest: "x",
					},
				},
				events: events,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalidJSON, default set -> success",
			args: args{
				jsonObj: []byte(``),
				params: []v1alpha1.ResourceParameter{
					v1alpha1.ResourceParameter{
						Src: &v1alpha1.ResourceParameterSource{
							Signal: "invalidJSON",
							Value:  &defaultValue,
						},
						Dest: "x",
					},
				},
				events: events,
			},
			want:    []byte(`{"x":"default"}`),
			wantErr: false,
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
