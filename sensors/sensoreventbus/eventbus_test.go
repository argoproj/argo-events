package sensoreventbus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testSubject    = "subject"
	testSensorName = "sensor-xxxxx"
)

var (
	testBadBusConfig = eventbusv1alpha1.BusConfig{}

	testValidSensorSpec = &v1alpha1.Sensor{
		metav1.TypeMeta{},
		metav1.ObjectMeta{Name: testSensorName},
		v1alpha1.SensorSpec{},
		v1alpha1.SensorStatus{},
	}

	testNoNameSensorSpec = &v1alpha1.Sensor{
		metav1.TypeMeta{},
		metav1.ObjectMeta{},
		v1alpha1.SensorSpec{},
		v1alpha1.SensorStatus{},
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

func TestGetDriver(t *testing.T) {
	t.Run("get driver without eventbus", func(t *testing.T) {
		_, err := GetDriver(context.Background(), testBadBusConfig, testValidSensorSpec)
		assert.Error(t, err)
	})

	t.Run("get driver with none auth eventbus", func(t *testing.T) {
		driver, err := GetDriver(context.Background(), testBusConfig, testValidSensorSpec)
		assert.NoError(t, err)
		assert.NotNil(t, driver)
	})

	t.Run("get driver with invalid sensor spec", func(t *testing.T) {
		_, err := GetDriver(context.Background(), testBusConfig, testNoNameSensorSpec)
		assert.Error(t, err)
	})

	t.Run("get driver with nil sensor spec", func(t *testing.T) {
		_, err := GetDriver(context.Background(), testBusConfig, nil)
		assert.Error(t, err)
	})

}
