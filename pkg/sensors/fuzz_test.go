/*
Copyright 2025 The Argoproj Authors.
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

package sensors

import (
	"context"
	"testing"

	fuzz "github.com/AdaLogics/go-fuzz-headers"
	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

func FuzzGetDependencyExpression(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		fdp := fuzz.NewConsumer(data)
		templ := &v1alpha1.TriggerTemplate{}
		err := fdp.GenerateStruct(templ)
		if err != nil {
			return
		}
		sensorObj := &v1alpha1.Sensor{}
		err = fdp.GenerateStruct(sensorObj)
		if err != nil {
			return
		}
		ft := &v1alpha1.Trigger{}
		err = fdp.GenerateStruct(ft)
		if err != nil {
			return
		}
		ft.Template = templ
		sensorObj.Spec.Triggers = []v1alpha1.Trigger{*ft}
		sensorCtx := &SensorContext{
			sensor: sensorObj,
		}
		_, _ = sensorCtx.getDependencyExpression(context.Background(), *ft)
	})
}
