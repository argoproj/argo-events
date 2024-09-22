package util

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
