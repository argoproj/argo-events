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
	"context"
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"strings"
)

var (
	configmapName = common.DefaultConfigMapName("gateway-transformer")
)

func TestWatchControllerConfigMap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tCtx := tOperationCtx{
		Config:        &tConfig{},
		Namespace:     "testing",
		kubeClientset: fake.NewSimpleClientset(),
	}
	_, err := tCtx.WatchGatewayTransformerConfigMap(ctx, configmapName)
	assert.Nil(t, err)
}

func TestNewControllerConfigMapWatch(t *testing.T) {
	tCtx := tOperationCtx{
		Config:        &tConfig{},
		Namespace:     "testing",
		kubeClientset: fake.NewSimpleClientset(),
	}
	watcher := tCtx.newStoreConfigMapWatch(configmapName)
	assert.NotNil(t, watcher)
}

func TestUpdateConfig(t *testing.T) {
	tCtx := tOperationCtx{
		Config: &tConfig{
			EventSource:      "test-source",
			EventType:        "test-type",
			EventTypeVersion: "test-0.1",
			Gateways: []v1alpha1.GatewayNotificationWatcher{
				{
					Name:     "test-gateway",
					Endpoint: "/test",
					Port:     "9000",
				},
			},
			Sensors: []v1alpha1.SensorNotificationWatcher{{
				Name: "test-sensor",
			}},
		},
		Namespace:     "testing",
		kubeClientset: fake.NewSimpleClientset(),
	}

	oldtCtx := tCtx

	gatewayWatchers := []string{`gateways:
    - name: "test-2"
      port: "9070"
      endpoint: "/test2"`}

	sensorWatchers := []string{`sensors:
    - name: "test-sensor2"`}

	updatedCm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configmapName,
			Namespace: "testing",
		},
		Data: map[string]string{
			common.EventSource:      "test-source-1",
			common.EventTypeVersion: "test-0.2",
			common.EventType:        string("test-type-1"),
			// list of sensor watchers
			common.SensorWatchers: strings.Join(sensorWatchers, ","),
			// list of gateway watchers
			common.GatewayWatchers: strings.Join(gatewayWatchers, ","),
		},
	}

	err := tCtx.updateConfig(updatedCm)
	assert.Nil(t, err)

	assert.NotEqual(t, tCtx.Config, oldtCtx.Config)
	assert.NotEqual(t, tCtx.Config.EventSource, oldtCtx.Config.EventSource)
	assert.NotEqual(t, tCtx.Config.EventTypeVersion, oldtCtx.Config.EventTypeVersion)
	assert.NotEqual(t, tCtx.Config.EventType, oldtCtx.Config.EventType)
	assert.NotEqual(t, tCtx.Config.Sensors, oldtCtx.Config.Sensors)
	assert.NotEqual(t, tCtx.Config.Gateways, oldtCtx.Config.Gateways)
}

var controllerConfig = `
instanceID: argo-events
executorImage: argoproj/gateway-controller:latest
`
