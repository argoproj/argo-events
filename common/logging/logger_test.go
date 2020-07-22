/*
Copyright 2020 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logging

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/smartystreets/goconvey/convey"

	"github.com/argoproj/argo-events/common"
)

func TestNewArgoEventsLogger(t *testing.T) {
	convey.Convey("Get a logger", t, func() {
		log := NewArgoEventsLogger()
		convey.So(log, convey.ShouldNotBeNil)
		convey.So(log.GetLevel(), convey.ShouldEqual, logrus.InfoLevel)

		convey.Convey("If debug env var is set then log level must be at debug", func() {
			err := os.Setenv(common.EnvVarDebugLog, "true")
			convey.So(err, convey.ShouldBeNil)

			log = NewArgoEventsLogger()
			convey.So(log.GetLevel(), convey.ShouldEqual, logrus.DebugLevel)
		})
	})
}
