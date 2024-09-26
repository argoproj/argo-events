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
