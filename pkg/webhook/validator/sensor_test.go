package validator

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aev1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
)

var (
	fakeBus = &aev1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: aev1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      aev1.DefaultEventBusName,
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
	fakeSensorWithFinalizer = &aev1.Sensor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: aev1.SchemeGroupVersion.String(),
			Kind:       "Sensor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  testNamespace,
			Name:       "test-sensor",
			Generation: 1,
			Finalizers: []string{"test-finalizer"},
		},
		Spec: aev1.SensorSpec{
			Dependencies: []aev1.EventDependency{
				{
					Name:            "test-dep",
					EventSourceName: "test-source",
					EventName:       "test-event",
				},
			},
			Triggers: []aev1.Trigger{
				{
					Template: &aev1.TriggerTemplate{
						Name: "test-trigger",
						K8s: &aev1.StandardK8STrigger{
							Operation: aev1.Create,
							Source: &aev1.ArtifactLocation{
								Resource: &aev1.K8SResource{
									Value: []byte(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"test-trigger"}}`),
								},
							},
						},
					},
				},
			},
		},
	}
)

func TestValidateSensor(t *testing.T) {
	dir := "../../../examples/sensors"
	dirEntries, err := os.ReadDir(dir)
	assert.Nil(t, err)

	testBus := fakeBus.DeepCopy()
	testBus.Status.MarkDeployed("test", "test")
	testBus.Status.MarkConfigured()

	// Try to create the EventBus, but ignore "already exists" errors
	_, err = fakeEventsClient.EventBus(testNamespace).Create(context.TODO(), testBus, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		assert.Nil(t, err)
	}

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

// TestValidateSensorUpdateSameGeneration tests that ValidateUpdate allows metadata-only changes, such as finalizer removal
func TestValidateSensorUpdateSameGeneration(t *testing.T) {
	testSensor := fakeSensorWithFinalizer.DeepCopy()
	testSensor.Finalizers = []string{}
	testSensor.Labels = map[string]string{"test": "label"}

	assert.Equal(t, fakeSensorWithFinalizer.Generation, testSensor.Generation, "Test setup: generations should be the same")

	v := NewSensorValidator(fakeK8sClient, nil, nil, nil, fakeSensorWithFinalizer, testSensor)
	r := v.ValidateUpdate(contextWithLogger(t))

	assert.True(t, r.Allowed, "ValidateUpdate should allow changes when generation is unchanged")
	assert.Nil(t, r.Result, "ValidateUpdate should return nil result for allowed requests")
}

// TestValidateSensorUpdateNewGeneration tests that ValidateUpdate calls ValidateCreate when Generation is different
func TestValidateSensorUpdateNewGeneration(t *testing.T) {
	testSensor := fakeSensorWithFinalizer.DeepCopy()
	testSensor.Finalizers = []string{}
	testSensor.Generation++

	assert.NotEqual(t, fakeSensorWithFinalizer.Generation, testSensor.Generation, "Test setup: generations should be different")

	v := NewSensorValidator(fakeK8sClient, nil, nil, nil, fakeSensorWithFinalizer, testSensor)
	r := v.ValidateUpdate(contextWithLogger(t))

	assert.False(t, r.Allowed, "ValidateUpdate should deny when ValidateCreate is called with nil eventBusClient")
	assert.NotNil(t, r.Result, "ValidateUpdate should return result with error message")
	assert.Contains(t, r.Result.Message, "eventBusClient is nil", "Error message should indicate nil eventBusClient")
}

// TestValidateSensorUpdateNewGenerationValidation tests that ValidateUpdate calls ValidateCreate when generation changes
func TestValidateSensorUpdateNewGenerationValidation(t *testing.T) {
	testBus := fakeBus.DeepCopy()
	testBus.Status.MarkDeployed("test", "test")
	testBus.Status.MarkConfigured()

	// Try to create the EventBus, but ignore "already exists" errors
	_, err := fakeEventsClient.EventBus(testNamespace).Create(context.TODO(), testBus, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		assert.Nil(t, err)
	}

	testSensor := fakeSensorWithFinalizer.DeepCopy()
	testSensor.Finalizers = []string{}
	testSensor.Generation++

	v := NewSensorValidator(fakeK8sClient, fakeEventsClient.EventBus(testNamespace), fakeEventsClient.EventSources(testNamespace), fakeEventsClient.Sensors(testNamespace), fakeSensorWithFinalizer, testSensor)
	r := v.ValidateUpdate(contextWithLogger(t))

	assert.True(t, r.Allowed, "ValidateUpdate should succeed when generation changes and validation passes")
	assert.Nil(t, r.Result, "ValidateUpdate should return nil result for allowed requests")
}

// TestValidateSensorUpdateNewGenerationValidationFails tests that ValidateUpdate returns validation failure when generation changes and EventBus is missing
func TestValidateSensorUpdateNewGenerationValidationFails(t *testing.T) {
	testSensor := fakeSensorWithFinalizer.DeepCopy()
	testSensor.Finalizers = []string{}
	testSensor.Generation++
	testSensor.Spec.EventBusName = "non-existent-eventbus"

	v := NewSensorValidator(fakeK8sClient, fakeEventsClient.EventBus(testNamespace), fakeEventsClient.EventSources(testNamespace), fakeEventsClient.Sensors(testNamespace), fakeSensorWithFinalizer, testSensor)
	r := v.ValidateUpdate(contextWithLogger(t))

	assert.False(t, r.Allowed, "ValidateUpdate should fail when generation changes and validation fails")
	assert.NotNil(t, r.Result, "ValidateUpdate should return a result with error message")
	assert.Contains(t, r.Result.Message, "failed to get EventBus", "Error message should mention EventBus failure")
}

// TestValidateSensorCreateDenied tests that ValidateCreate returns denied response when EventBus is not found
func TestValidateSensorCreateDenied(t *testing.T) {
	testSensor := fakeSensorWithFinalizer.DeepCopy()
	testSensor.Spec.EventBusName = "non-existent-eventbus"

	v := NewSensorValidator(fakeK8sClient, fakeEventsClient.EventBus(testNamespace), fakeEventsClient.EventSources(testNamespace), fakeEventsClient.Sensors(testNamespace), nil, testSensor)
	r := v.ValidateCreate(contextWithLogger(t))

	assert.False(t, r.Allowed, "ValidateCreate should deny when EventBus is not found")
	assert.NotNil(t, r.Result, "ValidateCreate should return a result with error message")
	assert.Contains(t, r.Result.Message, "failed to get EventBus", "Error message should mention EventBus failure")
}
