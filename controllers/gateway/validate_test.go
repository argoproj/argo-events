/*
Copyright 2018 BlackRock, Inc.

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

package gateway

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestValidate(t *testing.T) {
	convey.Convey("Given a gateway", t, func() {
		gateway, err := getGateway()

		convey.Convey("Make sure gateway is a valid gateway", func() {
			convey.So(err, convey.ShouldBeNil)
			convey.So(gateway, convey.ShouldNotBeNil)

			err := Validate(gateway)
			convey.So(err, convey.ShouldBeNil)
		})
	})
}
