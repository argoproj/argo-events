package sensor

import (
	"testing"

	eventbuscommon "github.com/argoproj/argo-events/eventbus/common"
	sensorv1alpha1 "github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestConsumerOptionsForDependency(t *testing.T) {
	conn := &JetstreamTriggerConn{}

	tests := []struct {
		name       string
		dependency eventbuscommon.Dependency
	}{
		{
			name: "DeliverAll policy",
			dependency: eventbuscommon.Dependency{
				Name:      "test-dep",
				JetStream: &sensorv1alpha1.JetStreamConsumerConfig{DeliverPolicy: sensorv1alpha1.JetStreamDeliverAll},
			},
		},
		{
			name: "DeliverLast policy",
			dependency: eventbuscommon.Dependency{
				Name:      "test-dep",
				JetStream: &sensorv1alpha1.JetStreamConsumerConfig{DeliverPolicy: sensorv1alpha1.JetStreamDeliverLast},
			},
		},
		{
			name: "DeliverNew policy",
			dependency: eventbuscommon.Dependency{
				Name:      "test-dep",
				JetStream: &sensorv1alpha1.JetStreamConsumerConfig{DeliverPolicy: sensorv1alpha1.JetStreamDeliverNew},
			},
		},
		{
			name:       "No JetStream config",
			dependency: eventbuscommon.Dependency{Name: "test-dep"},
		},
		{
			name: "Empty deliver policy",
			dependency: eventbuscommon.Dependency{
				Name:      "test-dep",
				JetStream: &sensorv1alpha1.JetStreamConsumerConfig{DeliverPolicy: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := conn.consumerOptionsForDependency(tt.dependency)
			assert.NotEmpty(t, opts)
			assert.GreaterOrEqual(t, len(opts), 1)
		})
	}
}

func TestResolveJetStreamDeliverConfig(t *testing.T) {
	conn := &JetstreamTriggerConn{}

	tests := []struct {
		name           string
		dependency     eventbuscommon.Dependency
		expectedPolicy sensorv1alpha1.JetStreamDeliverPolicy
		expectWarning  bool
	}{
		{
			name: "DeliverAll policy",
			dependency: eventbuscommon.Dependency{
				Name:      "test-dep",
				JetStream: &sensorv1alpha1.JetStreamConsumerConfig{DeliverPolicy: sensorv1alpha1.JetStreamDeliverAll},
			},
			expectedPolicy: sensorv1alpha1.JetStreamDeliverAll,
		},
		{
			name: "DeliverLast policy",
			dependency: eventbuscommon.Dependency{
				Name:      "test-dep",
				JetStream: &sensorv1alpha1.JetStreamConsumerConfig{DeliverPolicy: sensorv1alpha1.JetStreamDeliverLast},
			},
			expectedPolicy: sensorv1alpha1.JetStreamDeliverLast,
		},
		{
			name: "DeliverNew policy",
			dependency: eventbuscommon.Dependency{
				Name:      "test-dep",
				JetStream: &sensorv1alpha1.JetStreamConsumerConfig{DeliverPolicy: sensorv1alpha1.JetStreamDeliverNew},
			},
			expectedPolicy: sensorv1alpha1.JetStreamDeliverNew,
		},
		{
			name:           "No JetStream config",
			dependency:     eventbuscommon.Dependency{Name: "test-dep"},
			expectedPolicy: sensorv1alpha1.JetStreamDeliverNew,
		},
		{
			name: "Empty deliver policy",
			dependency: eventbuscommon.Dependency{
				Name:      "test-dep",
				JetStream: &sensorv1alpha1.JetStreamConsumerConfig{DeliverPolicy: ""},
			},
			expectedPolicy: sensorv1alpha1.JetStreamDeliverNew,
		},
		{
			name: "Unsupported policy",
			dependency: eventbuscommon.Dependency{
				Name:      "test-dep",
				JetStream: &sensorv1alpha1.JetStreamConsumerConfig{DeliverPolicy: "unsupported"},
			},
			expectedPolicy: sensorv1alpha1.JetStreamDeliverNew,
			expectWarning:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, warning := conn.resolveJetStreamDeliverConfig(tt.dependency)
			assert.Equal(t, tt.expectedPolicy, config.policy)

			if tt.expectWarning {
				assert.NotEmpty(t, warning)
			} else {
				assert.Empty(t, warning)
			}
		})
	}
}

func TestJetStreamDeliverPolicyConstants(t *testing.T) {
	assert.Equal(t, "all", string(sensorv1alpha1.JetStreamDeliverAll))
	assert.Equal(t, "last", string(sensorv1alpha1.JetStreamDeliverLast))
	assert.Equal(t, "new", string(sensorv1alpha1.JetStreamDeliverNew))
}

func TestJetStreamConsumerConfig(t *testing.T) {
	config := &sensorv1alpha1.JetStreamConsumerConfig{
		DeliverPolicy: sensorv1alpha1.JetStreamDeliverAll,
	}
	assert.Equal(t, sensorv1alpha1.JetStreamDeliverAll, config.DeliverPolicy)

	emptyConfig := &sensorv1alpha1.JetStreamConsumerConfig{}
	assert.Equal(t, sensorv1alpha1.JetStreamDeliverPolicy(""), emptyConfig.DeliverPolicy)
}
