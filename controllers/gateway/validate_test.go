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
	"fmt"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestValidate(t *testing.T) {
	dir := "../../examples/gateways"
	convey.Convey("Validate list of gateways", t, func() {
		files, err := ioutil.ReadDir(dir)
		convey.So(err, convey.ShouldBeNil)
		for _, file := range files {
			if strings.HasSuffix(file.Name(), "configmap.yaml") {
				continue
			}
			fmt.Println("filename: ", file.Name())
			content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", dir, file.Name()))
			convey.So(err, convey.ShouldBeNil)
			var gateway *v1alpha1.Gateway
			err = yaml.Unmarshal([]byte(content), &gateway)
			convey.So(err, convey.ShouldBeNil)
			err = Validate(gateway)
			convey.So(err, convey.ShouldBeNil)
		}
	})
}
