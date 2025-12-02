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

func TestParseTimeInTimezone(t *testing.T) {
	base := time.Date(2024, 7, 15, 14, 30, 0, 0, time.UTC)

	t.Run("parse time in UTC (empty timezone)", func(t *testing.T) {
		parsed, err := ParseTimeInTimezone("12:34:56", base, "")
		assert.NoError(t, err)
		assert.Equal(t, "12:34:56", parsed.Format("15:04:05"))
		assert.Equal(t, time.UTC, parsed.Location())
	})

	t.Run("parse time in America/New_York", func(t *testing.T) {
		parsed, err := ParseTimeInTimezone("09:00:00", base, "America/New_York")
		assert.NoError(t, err)

		// Verify the time is 9 AM in New York timezone
		assert.Equal(t, "09:00:00", parsed.Format("15:04:05"))

		loc, _ := time.LoadLocation("America/New_York")
		assert.Equal(t, loc.String(), parsed.Location().String())
	})

	t.Run("parse time in Europe/London", func(t *testing.T) {
		parsed, err := ParseTimeInTimezone("17:30:00", base, "Europe/London")
		assert.NoError(t, err)

		assert.Equal(t, "17:30:00", parsed.Format("15:04:05"))

		loc, _ := time.LoadLocation("Europe/London")
		assert.Equal(t, loc.String(), parsed.Location().String())
	})

	t.Run("parse time in Asia/Tokyo", func(t *testing.T) {
		parsed, err := ParseTimeInTimezone("23:59:59", base, "Asia/Tokyo")
		assert.NoError(t, err)

		assert.Equal(t, "23:59:59", parsed.Format("15:04:05"))

		loc, _ := time.LoadLocation("Asia/Tokyo")
		assert.Equal(t, loc.String(), parsed.Location().String())
	})

	t.Run("invalid timezone", func(t *testing.T) {
		_, err := ParseTimeInTimezone("12:00:00", base, "Invalid/Timezone")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid timezone")
	})

	t.Run("invalid time format", func(t *testing.T) {
		_, err := ParseTimeInTimezone("25:00:00", base, "America/New_York")
		assert.Error(t, err)
	})
}
