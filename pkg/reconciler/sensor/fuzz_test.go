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

package sensor

import (
	"context"
	"sync"
	"testing"

	fuzz "github.com/AdaLogics/go-fuzz-headers"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

var initter sync.Once

func initFunc() {
	_ = eventbusv1alpha1.AddToScheme(scheme.Scheme)
	_ = appv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)
}

func FuzzSensorControllerreconcile(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		initter.Do(initFunc)
		testImage := "test-image"
		f := fuzz.NewConsumer(data)
		sensorObj := &eventbusv1alpha1.Sensor{}
		err := f.GenerateStruct(sensorObj)
		if err != nil {
			return
		}
		ctx := context.Background()
		cl := fake.NewClientBuilder().Build()
		r := &reconciler{
			client:      cl,
			scheme:      scheme.Scheme,
			sensorImage: testImage,
			logger:      logging.NewArgoEventsLogger(),
		}
		_ = r.reconcile(ctx, sensorObj)
	})
}

func FuzzSensorControllerReconcile(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		initter.Do(initFunc)
		testImage := "test-image"
		f := fuzz.NewConsumer(data)
		testBus := &eventbusv1alpha1.EventBus{}
		err := f.GenerateStruct(testBus)
		if err != nil {
			return
		}
		sensorObj := &eventbusv1alpha1.Sensor{}
		err = f.GenerateStruct(sensorObj)
		if err != nil {
			return
		}
		testLabels := make(map[string]string)
		err = f.FuzzMap(&testLabels)
		if err != nil {
			return
		}
		ctx := context.Background()
		cl := fake.NewClientBuilder().Build()
		err = cl.Create(ctx, testBus)
		if err != nil {
			return
		}
		args := &AdaptorArgs{
			Image:  testImage,
			Sensor: sensorObj,
			Labels: testLabels,
		}
		_ = Reconcile(cl, nil, args, logging.NewArgoEventsLogger())
	})
}

func FuzzValidateSensor(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		fdp := fuzz.NewConsumer(data)
		eventBus := &eventbusv1alpha1.EventBus{}
		err := fdp.GenerateStruct(eventBus)
		if err != nil {
			return
		}
		sensor := &eventbusv1alpha1.Sensor{}
		err = fdp.GenerateStruct(sensor)
		if err != nil {
			return
		}
		_ = ValidateSensor(sensor, eventBus)
	})
}
