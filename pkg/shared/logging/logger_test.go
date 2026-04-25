package logging

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
	"go.uber.org/zap"
)

func TestNewArgoEventsLogger(t *testing.T) {
	convey.Convey("Get a logger", t, func() {
		log := NewArgoEventsLogger()
		convey.So(log, convey.ShouldNotBeNil)
	})
}

func TestConfigureLogLevelLogger(t *testing.T) {
	cases := map[string]zap.AtomicLevel{
		InfoLevel:  zap.NewAtomicLevelAt(zap.InfoLevel),
		DebugLevel: zap.NewAtomicLevelAt(zap.DebugLevel),
		ErrorLevel: zap.NewAtomicLevelAt(zap.ErrorLevel),
		WarnLevel:  zap.NewAtomicLevelAt(zap.WarnLevel),
		"":         zap.NewAtomicLevelAt(zap.InfoLevel),
		"garbage":  zap.NewAtomicLevelAt(zap.InfoLevel),
	}
	for level, want := range cases {
		t.Run(level, func(t *testing.T) {
			got := ConfigureLogLevelLogger(level).Level
			if got.Level() != want.Level() {
				t.Fatalf("ConfigureLogLevelLogger(%q).Level = %v, want %v", level, got.Level(), want.Level())
			}
		})
	}
}
