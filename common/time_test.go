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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseTime(t *testing.T) {
	timeStr := "12:34:56"
	base := time.Date(2020, 7, 8, 1, 2, 03, 123456789, time.FixedZone("UTC+9", 9*60*60))

	parsed, err := ParseTime(timeStr, base)
	if assert.NoError(t, err) {
		assert.Equal(t, timeStr, parsed.Format("15:04:05"))
		assert.Zero(t, parsed.Nanosecond())
		assert.Equal(t, base.UTC().Truncate(24*time.Hour), parsed.Truncate(24*time.Hour))
	}
}
