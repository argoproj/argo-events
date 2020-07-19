/*
Copyright 2020 BlackRock, Inc.

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
	"fmt"
	"time"
)

// ParseTime parses time string in "HH:MM:SS" format into time.Time, which date is same as baseDate in UTC.
func ParseTime(t string, baseDate time.Time) (time.Time, error) {
	date := baseDate.UTC().Format("2006-01-02")
	return time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s %s", date, t))
}
