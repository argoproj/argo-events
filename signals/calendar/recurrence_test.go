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

package calendar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseExclusionDate(t *testing.T) {
	exDateString := "EXDATE:20060102T150405Z,20180510T021030Z"

	dates := parseExclusionDates([]string{exDateString})
	assert.Equal(t, 2, len(dates))

	first := dates[0]
	assert.Equal(t, 2006, first.Year())
	assert.Equal(t, time.January, first.Month())
	assert.Equal(t, 2, first.Day())
	assert.Equal(t, 15, first.Hour())
	assert.Equal(t, 4, first.Minute())
	assert.Equal(t, 5, first.Second())

	second := dates[1]
	assert.Equal(t, 2018, second.Year())
	assert.Equal(t, time.May, second.Month())
	assert.Equal(t, 10, second.Day())
	assert.Equal(t, 2, second.Hour())
	assert.Equal(t, 10, second.Minute())
	assert.Equal(t, 30, second.Second())
}
