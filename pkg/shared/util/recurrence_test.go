package util

import (
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

func TestParseExclusionDate(t *testing.T) {
	convey.Convey("Given an exclusion date", t, func() {
		exDateString := "EXDATE:20060102T150405Z,20180510T021030Z"

		convey.Convey("Parse exclusion dates", func() {
			dates, err := ParseExclusionDates([]string{exDateString})

			convey.Convey("Make sure no error occurs", func() {
				convey.So(err, convey.ShouldBeEmpty)

				convey.Convey("Make sure only two dates are parsed", func() {
					convey.So(len(dates), convey.ShouldEqual, 2)

					convey.Convey("The first date must be 2nd of January", func() {
						first := dates[0]
						convey.So(first.Year(), convey.ShouldEqual, 2006)
						convey.So(first.Month(), convey.ShouldEqual, time.January)
						convey.So(first.Day(), convey.ShouldEqual, 2)
						convey.So(first.Hour(), convey.ShouldEqual, 15)
						convey.So(first.Minute(), convey.ShouldEqual, 4)
						convey.So(first.Second(), convey.ShouldEqual, 5)
					})

					convey.Convey("The second date must be 10th of May", func() {
						second := dates[1]
						convey.So(second.Year(), convey.ShouldEqual, 2018)
						convey.So(second.Month(), convey.ShouldEqual, time.May)
						convey.So(second.Day(), convey.ShouldEqual, 10)
						convey.So(second.Hour(), convey.ShouldEqual, 2)
						convey.So(second.Minute(), convey.ShouldEqual, 10)
						convey.So(second.Second(), convey.ShouldEqual, 30)
					})
				})
			})
		})
	})
}
