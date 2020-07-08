package eventbus

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-events/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	testSubject  = "subject"
	testClientID = "client-xxxxx"
)

var (
	testBadBusConfig = eventbusv1alpha1.BusConfig{}

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
		_, err := GetDriver(testBadBusConfig, testSubject, testClientID, common.NewArgoEventsLogger())
		assert.Error(t, err)
	})

	t.Run("get driver with none auth eventbus", func(t *testing.T) {
		driver, err := GetDriver(testBusConfig, testSubject, testClientID, common.NewArgoEventsLogger())
		assert.NoError(t, err)
		assert.NotNil(t, driver)
	})
}
