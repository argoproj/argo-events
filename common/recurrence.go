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

package common

import (
	"strings"
	"time"
)

const (
	exceptionDateTimePrefix = "EXDATE:"
	dateTimeFormat          = "20060102T150405Z"
)

// ParseExclusionDates parses the exclusion dates from the vals string according to RFC 5545
func ParseExclusionDates(vals []string) ([]time.Time, error) {
	exclusionDates := make([]time.Time, 0)
	for _, val := range vals {
		if strings.HasPrefix(val, exceptionDateTimePrefix) {
			dates, err := parseDateTimes(strings.TrimPrefix(val, exceptionDateTimePrefix))
			if err != nil {
				return nil, err
			}
			for _, d := range dates {
				exclusionDates = append(exclusionDates, d)
			}
		}
	}
	return exclusionDates, nil
}

func parseDateTimes(s string) ([]time.Time, error) {
	res := make([]time.Time, 0)
	stringDates := strings.Split(s, ",")
	for _, stringDate := range stringDates {
		t, err := time.Parse(dateTimeFormat, stringDate)
		if err != nil {
			return nil, err
		}
		res = append(res, t)
	}
	return res, nil
}
