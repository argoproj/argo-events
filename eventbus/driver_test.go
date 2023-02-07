package eventbus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testSensorName      = "sensor-xxxxx"
	testEventSourceName = "es-xxxxx"
	testSubject         = "subj-xxxxx"
	testHostname        = "sensor-xxxxx-xxxxx"
)

var (
	testBadBusConfig = eventbusv1alpha1.BusConfig{}

	testValidSensorSpec = &v1alpha1.Sensor{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: testSensorName},
		Spec:       v1alpha1.SensorSpec{},
		Status:     v1alpha1.SensorStatus{},
	}

	testNoNameSensorSpec = &v1alpha1.Sensor{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       v1alpha1.SensorSpec{},
		Status:     v1alpha1.SensorStatus{},
	}

	testClusterID = "test"
	testBusConfig = eventbusv1alpha1.BusConfig{
		NATS: &eventbusv1alpha1.NATSConfig{
			URL:       "nats://test:4222",
			ClusterID: &testClusterID,
			Auth:      &eventbusv1alpha1.AuthStrategyNone,
		},
	}
)

func TestGetSensorDriver(t *testing.T) {
	t.Run("get driver without eventbus", func(t *testing.T) {
		_, err := GetSensorDriver(context.Background(), testBadBusConfig, testValidSensorSpec, testHostname)
		assert.Error(t, err)
	})

	t.Run("get driver with none auth eventbus", func(t *testing.T) {
		driver, err := GetSensorDriver(context.Background(), testBusConfig, testValidSensorSpec, testHostname)
		assert.NoError(t, err)
		assert.NotNil(t, driver)
	})

	t.Run("get driver with invalid sensor spec", func(t *testing.T) {
		_, err := GetSensorDriver(context.Background(), testBusConfig, testNoNameSensorSpec, testHostname)
		assert.Error(t, err)
	})

	t.Run("get driver with nil sensor spec", func(t *testing.T) {
		_, err := GetSensorDriver(context.Background(), testBusConfig, nil, testHostname)
		assert.Error(t, err)
	})
}

func TestGetSourceDriver(t *testing.T) {
	t.Run("get driver without eventbus", func(t *testing.T) {
		_, err := GetEventSourceDriver(context.Background(), testBadBusConfig, testEventSourceName, testSubject)
		assert.Error(t, err)
	})

	t.Run("get driver with none auth eventbus", func(t *testing.T) {
		driver, err := GetEventSourceDriver(context.Background(), testBusConfig, testEventSourceName, testSubject)
		assert.NoError(t, err)
		assert.NotNil(t, driver)
	})

	t.Run("get driver without eventSourceName", func(t *testing.T) {
		_, err := GetEventSourceDriver(context.Background(), testBusConfig, "", testSubject)
		assert.Error(t, err)
	})

	t.Run("get NATS Streaming driver without subject", func(t *testing.T) {
		_, err := GetEventSourceDriver(context.Background(), testBusConfig, testEventSourceName, "")
		assert.Error(t, err)
	})
}
