package logging

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestNewArgoEventsLogger(t *testing.T) {
	convey.Convey("Get a logger", t, func() {
		log := NewArgoEventsLogger()
		convey.So(log, convey.ShouldNotBeNil)
	})
}
