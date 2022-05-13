package validator

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

var (
	fakeBus = &eventbusv1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: eventbusv1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      common.DefaultEventBusName,
		},
		Spec: eventbusv1alpha1.EventBusSpec{
			NATS: &eventbusv1alpha1.NATSBus{
				Native: &eventbusv1alpha1.NativeStrategy{
					Auth: &eventbusv1alpha1.AuthStrategyToken,
				},
			},
		},
		Status: eventbusv1alpha1.EventBusStatus{
			Config: eventbusv1alpha1.BusConfig{
				NATS: &eventbusv1alpha1.NATSConfig{
					URL:  "nats://xxxx",
					Auth: &eventbusv1alpha1.AuthStrategyToken,
					AccessSecret: &corev1.SecretKeySelector{
						Key: "test-key",
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-name",
						},
					},
				},
			},
		},
	}
)

func TestValidateSensor(t *testing.T) {
	dir := "../../examples/sensors"
	dirEntries, err := os.ReadDir(dir)
	assert.Nil(t, err)

	testBus := fakeBus.DeepCopy()
	testBus.Status.MarkDeployed("test", "test")
	testBus.Status.MarkConfigured()
	_, err = fakeEventBusClient.ArgoprojV1alpha1().EventBus(testNamespace).Create(context.TODO(), testBus, metav1.CreateOptions{})
	assert.Nil(t, err)

	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}
		content, err := os.ReadFile(fmt.Sprintf("%s/%s", dir, entry.Name()))
		assert.Nil(t, err)
		var sensor *v1alpha1.Sensor
		err = yaml.Unmarshal(content, &sensor)
		assert.Nil(t, err)
		sensor.Namespace = testNamespace
		newSensor := sensor.DeepCopy()
		newSensor.Generation++
		v := NewSensorValidator(fakeK8sClient, fakeEventBusClient, fakeEventSourceClient, fakeSensorClient, sensor, newSensor)
		r := v.ValidateCreate(contextWithLogger(t))
		assert.True(t, r.Allowed)
		r = v.ValidateUpdate(contextWithLogger(t))
		assert.True(t, r.Allowed)
	}
}
