package util

import (
	"strings"
	"testing"
	"time"

	cronlib "github.com/robfig/cron/v3"
)

func TestPrevCronTime(t *testing.T) {
	runs := []struct {
		time, spec  string
		expected    string
		expectedErr bool
	}{
		// Simple cases
		{"Mon Jul 9 15:00 2012", "0 0/15 * * * *", "Mon Jul 9 14:45 2012", false},
		{"Mon Jul 9 14:59 2012", "0 0/15 * * * *", "Mon Jul 9 14:45 2012", false},
		{"Mon Jul 9 15:01:59 2012", "0 0/15 * * * *", "Mon Jul 9 15:00 2012", false},

		// Wrap around hours
		{"Mon Jul 9 15:10 2012", "0 20-35/15 * * * *", "Mon Jul 9 14:35 2012", false},

		// Wrap around days
		{"Tue Jul 10 00:00 2012", "0 */15 * * * *", "Tue Jul 9 23:45 2012", false},
		{"Tue Jul 10 00:00 2012", "0 20-35/15 * * * *", "Tue Jul 9 23:35 2012", false},

		// Wrap around months
		{"Mon Jul 9 09:35 2012", "0 0 12 9 Apr-Oct ?", "Sat Jun 9 12:00 2012", false},

		// Leap year
		{"Mon Jul 9 23:35 2018", "0 0 0 29 Feb ?", "Mon Feb 29 00:00 2016", false},

		// Daylight savings time 3am EDT (-4) -> 2am EST (-5)
		{"2013-03-11T02:30:00-0400", "TZ=America/New_York 0 0 12 9 Mar ?", "2013-03-09T12:00:00-0500", false},

		// hourly job
		{"2012-03-11T01:00:00-0500", "TZ=America/New_York 0 0 * * * ?", "2012-03-11T00:00:00-0500", false},

		// 2am nightly job (skipped)
		{"2012-03-12T00:00:00-0400", "TZ=America/New_York 0 0 2 * * ?", "2012-03-10T02:00:00-0500", false},

		// 2am nightly job
		{"2012-11-04T02:00:00-0500", "TZ=America/New_York 0 0 0 * * ?", "2012-11-04T00:00:00-0400", false},
		{"2012-11-05T02:00:00-0500", "TZ=America/New_York 0 0 2 * * ?", "2012-11-04T02:00:00-0500", false},

		// Unsatisfiable
		{"Mon Jul 9 23:35 2012", "0 0 0 30 Feb ?", "", true},
		{"Mon Jul 9 23:35 2012", "0 0 0 31 Apr ?", "", true},

		// Monthly job
		{"TZ=America/New_York 2012-12-03T00:00:00-0500", "0 0 3 3 * ?", "2012-11-03T03:00:00-0400", false},
	}

	parser := cronlib.NewParser(cronlib.Second | cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.DowOptional | cronlib.Descriptor)

	for _, c := range runs {
		actual, err := PrevCronTime(c.spec, parser, getTime(c.time))
		if c.expectedErr {
			if err == nil {
				t.Errorf("%s, \"%s\": should have received error but didn't", c.time, c.spec)
			}
		} else {
			if err != nil {
				t.Errorf("%s, \"%s\": error: %v", c.time, c.spec, err)
			} else {
				expected := getTime(c.expected)
				if !actual.Equal(expected) {
					t.Errorf("%s, \"%s\": (expected) %v != %v (actual)", c.time, c.spec, expected, actual)
				}
			}
		}
	}
}

func getTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}

	var location = time.Local
	if strings.HasPrefix(value, "TZ=") {
		parts := strings.Fields(value)
		loc, err := time.LoadLocation(parts[0][len("TZ="):])
		if err != nil {
			panic("could not parse location:" + err.Error())
		}
		location = loc
		value = parts[1]
	}

	var layouts = []string{
		"Mon Jan 2 15:04 2006",
		"Mon Jan 2 15:04:05 2006",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, value, location); err == nil {
			return t
		}
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05-0700", value, location); err == nil {
		return t
	}
	panic("could not parse time value " + value)
}
