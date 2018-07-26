package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseExclusionDate(t *testing.T) {
	exDateString := "EXDATE:20060102T150405Z,20180510T021030Z"

	dates, err := ParseExclusionDates([]string{exDateString})
	if err != nil {
		t.Fatal(err)
	}
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
