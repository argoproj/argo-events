package dependencies

import (
	"testing"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
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

func TestFilterTimeWithTimezone(t *testing.T) {
	// Test date: July 15, 2024 (summer - DST in effect for US timezones)

	t.Run("UTC timezone (default)", func(t *testing.T) {
		// Event at 10:00 UTC
		eventTime := time.Date(2024, 7, 15, 10, 0, 0, 0, time.UTC)

		timeFilter := &v1alpha1.TimeFilter{
			Start: "09:00:00",
			Stop:  "17:00:00",
			// No timezone specified, defaults to UTC
		}

		result, err := filterTime(timeFilter, eventTime)
		assert.NoError(t, err)
		assert.True(t, result, "10:00 UTC should be between 09:00 and 17:00 UTC")
	})

	t.Run("America/New_York timezone - event passes", func(t *testing.T) {
		// Event at 14:00 UTC = 10:00 EDT (Eastern Daylight Time)
		eventTime := time.Date(2024, 7, 15, 14, 0, 0, 0, time.UTC)

		timeFilter := &v1alpha1.TimeFilter{
			Start:    "09:00:00",
			Stop:     "17:00:00",
			Timezone: "America/New_York",
		}

		result, err := filterTime(timeFilter, eventTime)
		assert.NoError(t, err)
		assert.True(t, result, "14:00 UTC (10:00 EDT) should be between 09:00 and 17:00 EDT")
	})

	t.Run("America/New_York timezone - event fails (too early)", func(t *testing.T) {
		// Event at 12:00 UTC = 08:00 EDT
		eventTime := time.Date(2024, 7, 15, 12, 0, 0, 0, time.UTC)

		timeFilter := &v1alpha1.TimeFilter{
			Start:    "09:00:00",
			Stop:     "17:00:00",
			Timezone: "America/New_York",
		}

		result, err := filterTime(timeFilter, eventTime)
		assert.NoError(t, err)
		assert.False(t, result, "12:00 UTC (08:00 EDT) should be before 09:00 EDT")
	})

	t.Run("America/New_York timezone - event fails (too late)", func(t *testing.T) {
		// Event at 22:00 UTC = 18:00 EDT
		eventTime := time.Date(2024, 7, 15, 22, 0, 0, 0, time.UTC)

		timeFilter := &v1alpha1.TimeFilter{
			Start:    "09:00:00",
			Stop:     "17:00:00",
			Timezone: "America/New_York",
		}

		result, err := filterTime(timeFilter, eventTime)
		assert.NoError(t, err)
		assert.False(t, result, "22:00 UTC (18:00 EDT) should be after 17:00 EDT")
	})

	t.Run("Europe/London timezone", func(t *testing.T) {
		// Event at 10:00 UTC = 11:00 BST (British Summer Time)
		eventTime := time.Date(2024, 7, 15, 10, 0, 0, 0, time.UTC)

		timeFilter := &v1alpha1.TimeFilter{
			Start:    "10:00:00",
			Stop:     "18:00:00",
			Timezone: "Europe/London",
		}

		result, err := filterTime(timeFilter, eventTime)
		assert.NoError(t, err)
		assert.True(t, result, "10:00 UTC (11:00 BST) should be between 10:00 and 18:00 BST")
	})

	t.Run("Asia/Tokyo timezone", func(t *testing.T) {
		// Event at 01:00 UTC = 10:00 JST
		eventTime := time.Date(2024, 7, 15, 1, 0, 0, 0, time.UTC)

		timeFilter := &v1alpha1.TimeFilter{
			Start:    "09:00:00",
			Stop:     "17:00:00",
			Timezone: "Asia/Tokyo",
		}

		result, err := filterTime(timeFilter, eventTime)
		assert.NoError(t, err)
		assert.True(t, result, "01:00 UTC (10:00 JST) should be between 09:00 and 17:00 JST")
	})

	t.Run("invalid timezone", func(t *testing.T) {
		eventTime := time.Date(2024, 7, 15, 10, 0, 0, 0, time.UTC)

		timeFilter := &v1alpha1.TimeFilter{
			Start:    "09:00:00",
			Stop:     "17:00:00",
			Timezone: "Invalid/Timezone",
		}

		result, err := filterTime(timeFilter, eventTime)
		assert.Error(t, err)
		assert.False(t, result)
		assert.Contains(t, err.Error(), "invalid timezone")
	})

	t.Run("overnight window with timezone", func(t *testing.T) {
		// Event at 02:00 UTC = 22:00 EDT (previous day in ET)
		eventTime := time.Date(2024, 7, 15, 2, 0, 0, 0, time.UTC)

		timeFilter := &v1alpha1.TimeFilter{
			Start:    "22:00:00",
			Stop:     "06:00:00",
			Timezone: "America/New_York",
		}

		result, err := filterTime(timeFilter, eventTime)
		assert.NoError(t, err)
		assert.True(t, result, "02:00 UTC (22:00 EDT) should be in overnight window [22:00, 06:00)")
	})

	t.Run("winter timezone - America/New_York EST", func(t *testing.T) {
		// Test date: January 15, 2024 (winter - EST in effect)
		// Event at 15:00 UTC = 10:00 EST
		eventTime := time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC)

		timeFilter := &v1alpha1.TimeFilter{
			Start:    "09:00:00",
			Stop:     "17:00:00",
			Timezone: "America/New_York",
		}

		result, err := filterTime(timeFilter, eventTime)
		assert.NoError(t, err)
		assert.True(t, result, "15:00 UTC (10:00 EST) should be between 09:00 and 17:00 EST")
	})
}
