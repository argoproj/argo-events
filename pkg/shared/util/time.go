package util

import (
	"fmt"
	"time"
)

// ParseTime parses time string in "HH:MM:SS" format into time.Time, which date is same as baseDate in UTC.
func ParseTime(t string, baseDate time.Time) (time.Time, error) {
	date := baseDate.UTC().Format("2006-01-02")
	return time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s %s", date, t))
}

// ParseTimeInTimezone parses time string in "HH:MM:SS" format into time.Time with the specified timezone.
// The date is same as baseDate, but in the specified timezone.
// If timezone is empty, defaults to UTC.
func ParseTimeInTimezone(t string, baseDate time.Time, timezone string) (time.Time, error) {
	loc := time.UTC
	var err error

	if timezone != "" {
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid timezone '%s': %w", timezone, err)
		}
	}

	// Convert baseDate to the specified timezone
	baseDateInTZ := baseDate.In(loc)
	date := baseDateInTZ.Format("2006-01-02")

	// Parse the time in the specified timezone
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", fmt.Sprintf("%s %s", date, t), loc)
	if err != nil {
		return time.Time{}, err
	}

	return parsed, nil
}
