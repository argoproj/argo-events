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

package transform

import (
	"bytes"
	"encoding/json"
	"github.com/argoproj/argo-events/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"net/http"
	"testing"
)

type ClosingBuffer struct {
	*bytes.Buffer
}

func (cb *ClosingBuffer) Close() error {
	return nil
}

func TestTOperationCtx_TransformRequest(t *testing.T) {
	tConfig := NewTransformerConfig("test", "test-1", "test-src", nil, nil)
	fakeClient := fake.NewSimpleClientset()
	tCtx := NewTransformOperationContext(tConfig, "testing", fakeClient)

	tp := &TransformerPayload{
		Src:     "test-gateway",
		Payload: []byte("this is payload"),
	}

	tpBytes, err := json.Marshal(tp)
	assert.Nil(t, err)

	request := http.Request{
		Body: &ClosingBuffer{
			bytes.NewBuffer(tpBytes),
		},
	}

	event, err := tCtx.transform(&request)
	assert.Nil(t, err)

	assert.Equal(t, event.Context.EventType, "test")
	assert.Equal(t, event.Context.EventTypeVersion, "test-1")
	assert.Equal(t, event.Context.Source.Host, "test-src"+"/"+"test-gateway")
	assert.Equal(t, string(event.Payload), "this is payload")

	fakeService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "testing",
			Name:      common.DefaultSensorServiceName("fake-sensor"),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:       intstr.Parse(common.SensorServicePort).IntVal,
					TargetPort: intstr.FromInt(int(intstr.Parse(common.SensorServicePort).IntVal)),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				common.LabelSensorName: "test-sensor",
			},
		},
	}

	svc, err := fakeClient.CoreV1().Services(tCtx.Namespace).Create(fakeService)
	assert.Nil(t, err)

	svcName, err := tCtx.getWatcherIP("fake-sensor")
	assert.Nil(t, err)
	assert.Equal(t, svcName, svc.ObjectMeta.Name)
}
