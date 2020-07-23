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

package sensor

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
)

func TestValidateSensor(t *testing.T) {
	// TODO: Temporarily skip it
	t.SkipNow()
	dir := "../../examples/sensors"
	files, err := ioutil.ReadDir(dir)
	assert.Nil(t, err)
	for _, file := range files {
		content, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", dir, file.Name()))
		assert.Nil(t, err)
		var sensor *v1alpha1.Sensor
		err = yaml.Unmarshal(content, &sensor)
		assert.Nil(t, err)
		err = ValidateSensor(sensor)
		assert.Nil(t, err)
	}
}
