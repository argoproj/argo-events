/*
Copyright 2026 The Argoproj Authors.

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

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	httptrigger "github.com/argoproj/argo-events/pkg/sensors/triggers/http"
	"github.com/argoproj/argo-events/pkg/shared/logging"
)

func TestGetTrigger_HTTPLogLevel(t *testing.T) {
	httpTrigger := &v1alpha1.Trigger{
		Template: &v1alpha1.TriggerTemplate{
			Name: "http-trigger",
			HTTP: &v1alpha1.HTTPTrigger{
				URL: "http://example.com",
			},
		},
	}

	t.Run("uses sensor log level for HTTP triggers", func(t *testing.T) {
		sensor := &v1alpha1.Sensor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-sensor",
				Namespace: "default",
			},
			Spec: v1alpha1.SensorSpec{
				LogLevel: logging.WarnLevel,
				Triggers: []v1alpha1.Trigger{*httpTrigger},
			},
		}

		sensorCtx := NewSensorContext(nil, nil, sensor, nil, "", "", nil)
		trigger := sensorCtx.GetTrigger(context.Background(), httpTrigger)
		assert.NotNil(t, trigger)

		httpT, ok := trigger.(*httptrigger.HTTPTrigger)
		assert.True(t, ok)
		core := httpT.Logger.Desugar().Core()
		assert.False(t, core.Enabled(zapcore.InfoLevel))
		assert.True(t, core.Enabled(zapcore.WarnLevel))
	})

	t.Run("keeps context logger level when log level is unset", func(t *testing.T) {
		sensor := &v1alpha1.Sensor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-sensor",
				Namespace: "default",
			},
			Spec: v1alpha1.SensorSpec{
				Triggers: []v1alpha1.Trigger{*httpTrigger},
			},
		}

		sensorCtx := NewSensorContext(nil, nil, sensor, nil, "", "", nil)
		ctx := logging.WithLogger(context.Background(), logging.NewSugaredLoggerWithLevel(logging.ErrorLevel))
		trigger := sensorCtx.GetTrigger(ctx, httpTrigger)
		assert.NotNil(t, trigger)

		httpT, ok := trigger.(*httptrigger.HTTPTrigger)
		assert.True(t, ok)
		core := httpT.Logger.Desugar().Core()
		assert.False(t, core.Enabled(zapcore.InfoLevel))
		assert.True(t, core.Enabled(zapcore.ErrorLevel))
	})
}
