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

package eventsource

import (
	"context"
	"sync"
	"testing"

	fuzz "github.com/AdaLogics/go-fuzz-headers"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

var initter sync.Once

func initScheme() {
	_ = v1alpha1.AddToScheme(scheme.Scheme)
	_ = appv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)
}

func FuzzEventsourceReconciler(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		initter.Do(initScheme)
		f := fuzz.NewConsumer(data)
		testEventSource := &v1alpha1.EventSource{}
		err := f.GenerateStruct(testEventSource)
		if err != nil {
			return
		}
		cl := fake.NewClientBuilder().Build()
		testStreamingImage := "test-steaming-image"
		r := &reconciler{
			client:           cl,
			scheme:           scheme.Scheme,
			eventSourceImage: testStreamingImage,
			logger:           logging.NewArgoEventsLogger(),
		}
		ctx := context.Background()
		_ = r.reconcile(ctx, testEventSource)
	})
}

func FuzzResourceReconcile(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		testImage := "test-image"
		initter.Do(initScheme)
		f := fuzz.NewConsumer(data)
		testEventSource := &v1alpha1.EventSource{}
		err := f.GenerateStruct(testEventSource)
		if err != nil {
			return
		}
		testLabels := make(map[string]string)
		err = f.FuzzMap(&testLabels)
		if err != nil {
			return
		}
		testBus := &v1alpha1.EventBus{}
		err = f.GenerateStruct(testBus)
		if err != nil {
			return
		}
		cl := fake.NewClientBuilder().Build()
		err = cl.Create(context.Background(), testBus)
		if err != nil {
			return
		}
		args := &AdaptorArgs{
			Image:       testImage,
			EventSource: testEventSource,
			Labels:      testLabels,
		}
		_ = Reconcile(cl, args, logging.NewArgoEventsLogger())
	})
}
