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

package aws_sqs

import (
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/gateways"
	"github.com/argoproj/argo-events/pkg/apis/eventsources/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListenEvents(t *testing.T) {
	convey.Convey("Given an event source, listen to events", t, func() {
		ese := &SQSEventSourceListener{
			Clientset: fake.NewSimpleClientset(),
			Namespace: "fake",
			Log:       common.NewArgoEventsLogger(),
		}

		dataCh := make(chan []byte)
		errorCh := make(chan error)
		doneCh := make(chan struct{}, 1)
		errorCh2 := make(chan error)

		go func() {
			err := <-errorCh
			errorCh2 <- err
		}()

		sqsEventSource := &v1alpha1.SQSEventSource{
			Region:          "us-east-1",
			Queue:           "test-queue",
			WaitTimeSeconds: 10,
			AccessKey: &corev1.SecretKeySelector{
				Key: "accessKey",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "sns",
				},
			},
			SecretKey: &corev1.SecretKeySelector{
				Key: "secretKey",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "sns",
				},
			},
		}

		body, err := yaml.Marshal(sqsEventSource)
		convey.So(err, convey.ShouldBeNil)

		ese.listenEvents(&gateways.EventSource{
			Name:    "fake",
			Value:   body,
			Id:      "1234",
			Type:    string(v1alpha1.SQSEvent),
			Version: "v0.10",
		}, dataCh, errorCh, doneCh)

		err = <-errorCh2
		convey.So(err, convey.ShouldNotBeNil)
	})

	convey.Convey("Given an event source without AWS credentials, listen to events", t, func() {
		sqsEventSource := &v1alpha1.SQSEventSource{
			Region:          "us-east-1",
			Queue:           "test-queue",
			WaitTimeSeconds: 10,
		}

		body, err := yaml.Marshal(sqsEventSource)
		convey.So(err, convey.ShouldBeNil)

		ese := &SQSEventSourceListener{
			Clientset: fake.NewSimpleClientset(),
			Namespace: "fake",
			Log:       common.NewArgoEventsLogger(),
		}

		dataCh := make(chan []byte)
		errorCh := make(chan error)
		doneCh := make(chan struct{}, 1)
		errorCh2 := make(chan error)

		go func() {
			err := <-errorCh
			errorCh2 <- err
		}()

		ese.listenEvents(&gateways.EventSource{
			Name:  "fake",
			Value: body,
			Id:    "1234",
		}, dataCh, errorCh, doneCh)

		err = <-errorCh2
		convey.So(err, convey.ShouldNotBeNil)
	})
}
