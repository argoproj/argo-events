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

package gateways

import (
	"github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestGatewayUtil(t *testing.T) {
	convey.Convey("Given a value, hash it", t, func() {
		hash := Hasher("test")
		convey.So(hash, convey.ShouldNotBeEmpty)
	})

	convey.Convey("Given a event source, set the validation message", t, func() {
		v := ValidEventSource{}
		SetValidEventSource(&v, "event source is valid", true)
		convey.So(v.IsValid, convey.ShouldBeTrue)
		convey.So(v.Reason, convey.ShouldEqual, "event source is valid")
	})
}
