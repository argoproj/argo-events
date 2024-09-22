package util

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
			exclusionDates = append(exclusionDates, dates...)
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
