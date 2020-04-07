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

package sensors

import (
	"context"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/argoproj/argo-events/sensors/types"
	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestHandleEvent(t *testing.T) {
	obj := sensorObj.DeepCopy()
	obj.Spec.Dependencies = []v1alpha1.EventDependency{
		{
			Name:        "dep1",
			GatewayName: "webhook-gateway",
			EventName:   "example-1",
		},
	}

	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetID("1")
	event.SetSource("webhook-gateway")
	event.SetSubject("example-1")
	event.SetType("webhook")
	event.SetDataContentType(common.MediaTypeJSON)
	event.SetTime(time.Now())

	queue := make(chan *types.Notification)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-queue:
			case <-done:
				return
			}
		}
	}()

	sensorCtx := &SensorContext{
		Sensor:            obj,
		NotificationQueue: queue,
		Logger:            common.NewArgoEventsLogger(),
	}

	tests := []struct {
		name       string
		updateFunc func()
		result     error
	}{
		{
			name:       "valid event is received",
			updateFunc: func() {},
			result:     nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.updateFunc()
			result := sensorCtx.handleEvent(context.Background(), event)
			assert.Equal(t, test.result, result)
		})
	}

	done <- struct{}{}
}
