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

package dependencies

import (
	"testing"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestFilterTime(t *testing.T) {
	now := time.Now().UTC()
	eventTimes := [6]time.Time{
		time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		time.Date(now.Year(), now.Month(), now.Day(), 4, 5, 6, 0, time.UTC),
		time.Date(now.Year(), now.Month(), now.Day(), 8, 9, 10, 0, time.UTC),
		time.Date(now.Year(), now.Month(), now.Day(), 12, 13, 14, 0, time.UTC),
		time.Date(now.Year(), now.Month(), now.Day(), 16, 17, 18, 0, time.UTC),
		time.Date(now.Year(), now.Month(), now.Day(), 20, 21, 22, 0, time.UTC),
	}

	time1 := eventTimes[2].Format("15:04:05")
	time2 := eventTimes[4].Format("15:04:05")

	tests := []struct {
		name       string
		timeFilter *v1alpha1.TimeFilter
		results    [6]bool
	}{
		{
			name:       "no filter",
			timeFilter: nil,
			results:    [6]bool{true, true, true, true, true, true},
			// With no filter, any event time should pass
		},
		{
			name: "start less than stop",
			timeFilter: &v1alpha1.TimeFilter{
				Start: time1,
				Stop:  time2,
			},
			results: [6]bool{false, false, true, true, false, false},
			//                             ~~~~~~~~~~
			//                            [time1     , time2)
		},
		{
			name: "stop less than start",
			timeFilter: &v1alpha1.TimeFilter{
				Start: time2,
				Stop:  time1,
			},
			results: [6]bool{true, true, false, false, true, true},
			//               ~~~~~~~~~~                ~~~~~~~~~~
			//              [          , time1)       [time2     , )
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for i, eventTime := range eventTimes {
				result, err := filterTime(test.timeFilter, eventTime)
				assert.Nil(t, err)
				assert.Equal(t, test.results[i], result)
			}
		})
	}
}
