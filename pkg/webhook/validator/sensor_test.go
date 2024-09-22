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

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

var (
	fakeBus = &aev1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: aev1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      sharedutil.DefaultEventBusName,
		},
		Spec: aev1.EventBusSpec{
			NATS: &aev1.NATSBus{
				Native: &aev1.NativeStrategy{
					Auth: &aev1.AuthStrategyToken,
				},
			},
		},
		Status: aev1.EventBusStatus{
			Config: aev1.BusConfig{
				NATS: &aev1.NATSConfig{
					URL:  "nats://xxxx",
					Auth: &aev1.AuthStrategyToken,
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
	_, err = fakeEventsClient.EventBus(testNamespace).Create(context.TODO(), testBus, metav1.CreateOptions{})
	assert.Nil(t, err)

	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}
		content, err := os.ReadFile(fmt.Sprintf("%s/%s", dir, entry.Name()))
		assert.Nil(t, err)
		var sensor *aev1.Sensor
		err = yaml.Unmarshal(content, &sensor)
		assert.Nil(t, err)
		sensor.Namespace = testNamespace
		newSensor := sensor.DeepCopy()
		newSensor.Generation++
		v := NewSensorValidator(fakeK8sClient, fakeEventsClient.EventBus(testNamespace), fakeEventsClient.EventSources(testNamespace), fakeEventsClient.Sensors(testNamespace), sensor, newSensor)
		r := v.ValidateCreate(contextWithLogger(t))
		assert.True(t, r.Allowed)
		r = v.ValidateUpdate(contextWithLogger(t))
		assert.True(t, r.Allowed)
	}
}
