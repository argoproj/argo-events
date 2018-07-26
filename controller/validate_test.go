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

package controller

import (
	"testing"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_validateSensor(t *testing.T) {
	type args struct {
		s *v1alpha1.Sensor
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid sensor",
			args: args{
				s: &v1alpha1.Sensor{
					Spec: v1alpha1.SensorSpec{
						Signals: []v1alpha1.Signal{
							v1alpha1.Signal{
								Name: "test-signal",
								Stream: &v1alpha1.Stream{
									Type: "test",
									URL:  "http://test.com",
								},
							},
						},
						Triggers: []v1alpha1.Trigger{
							v1alpha1.Trigger{
								Name:     "test-trigger",
								Resource: &v1alpha1.ResourceObject{},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateSensor(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("validateSensor() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateSignals(t *testing.T) {
	type args struct {
		signals []v1alpha1.Signal
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "no signals",
			args:    args{},
			wantErr: true,
		},
		{
			name: "signal w/o name",
			args: args{
				signals: []v1alpha1.Signal{v1alpha1.Signal{}},
			},
			wantErr: true,
		},
		{
			name: "unknown signal",
			args: args{
				signals: []v1alpha1.Signal{v1alpha1.Signal{Name: "test"}},
			},
			wantErr: true,
		},
		{
			name: "invalid stream - missing type",
			args: args{
				signals: []v1alpha1.Signal{v1alpha1.Signal{
					Stream: &v1alpha1.Stream{},
				}},
			},
			wantErr: true,
		},
		{
			name: "invalid stream - missing URL",
			args: args{
				signals: []v1alpha1.Signal{v1alpha1.Signal{
					Stream: &v1alpha1.Stream{
						Type: "test",
					},
				}},
			},
			wantErr: true,
		},
		{
			name: "valid stream",
			args: args{
				signals: []v1alpha1.Signal{v1alpha1.Signal{
					Name: "test-stream",
					Stream: &v1alpha1.Stream{
						Type: "test",
						URL:  "http://test.com",
					},
				}},
			},
		},
		{
			name: "invalid artifact - no location",
			args: args{
				signals: []v1alpha1.Signal{v1alpha1.Signal{
					Name:     "artifact-test",
					Artifact: &v1alpha1.ArtifactSignal{},
				}},
			},
			wantErr: true,
		},
		{
			name: "invalid artifact - invalid target stream",
			args: args{
				signals: []v1alpha1.Signal{v1alpha1.Signal{
					Artifact: &v1alpha1.ArtifactSignal{
						ArtifactLocation: v1alpha1.ArtifactLocation{
							File: &v1alpha1.FileArtifact{},
						},
						Target: v1alpha1.Stream{},
					},
				}},
			},
			wantErr: true,
		},
		{
			name: "invalid calendar - missing schedule",
			args: args{
				signals: []v1alpha1.Signal{v1alpha1.Signal{
					Calendar: &v1alpha1.CalendarSignal{},
				}},
			},
			wantErr: true,
		},
		{
			name: "invalid calendar - invalid recurrence",
			args: args{
				signals: []v1alpha1.Signal{v1alpha1.Signal{
					Calendar: &v1alpha1.CalendarSignal{
						Schedule:   "@every 5s",
						Recurrence: []string{"EXDATE:bla"},
					},
				}},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateSignals(tt.args.signals); (err != nil) != tt.wantErr {
				t.Errorf("validateSignals() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateSignalFilter(t *testing.T) {
	type args struct {
		f v1alpha1.SignalFilter
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "time filter - stop before start",
			args: args{
				f: v1alpha1.SignalFilter{
					Time: &v1alpha1.TimeFilter{
						Start: &metav1.Time{
							Time: time.Date(2018, time.May, 10, 0, 0, 0, 0, time.UTC),
						},
						Stop: &metav1.Time{
							Time: time.Date(2016, time.May, 10, 0, 0, 0, 0, time.UTC),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "time filter - stop before current time",
			args: args{
				f: v1alpha1.SignalFilter{
					Time: &v1alpha1.TimeFilter{
						Stop: &metav1.Time{
							Time: time.Now().UTC(),
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "time filter - both nil",
			args: args{
				f: v1alpha1.SignalFilter{
					Time: &v1alpha1.TimeFilter{},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateSignalFilter(tt.args.f); (err != nil) != tt.wantErr {
				t.Errorf("validateSignalFilter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
