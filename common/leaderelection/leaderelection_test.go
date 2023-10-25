package leaderelection

import (
	"context"
	"os"
	"testing"

	"github.com/argoproj/argo-events/common"
	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

var (
	configs = []eventbusv1alpha1.BusConfig{
		{NATS: &eventbusv1alpha1.NATSConfig{}},
		{JetStream: &eventbusv1alpha1.JetStreamConfig{}},
		{JetStream: &eventbusv1alpha1.JetStreamConfig{AccessSecret: &v1.SecretKeySelector{}}},
	}
)

func TestLeaderElectionWithInvalidEventBus(t *testing.T) {
	elector, err := NewElector(context.TODO(), eventbusv1alpha1.BusConfig{}, "", 0, "", "", "")

	assert.Nil(t, elector)
	assert.EqualError(t, err, "invalid event bus")
}

func TestLeaderElectionWithEventBusElector(t *testing.T) {
	eventBusAuthFileMountPath = "test"

	for _, config := range configs {
		elector, err := NewElector(context.TODO(), config, "", 0, "", "", "")
		assert.Nil(t, err)

		_, ok := elector.(*natsEventBusElector)
		assert.True(t, ok)
	}
}

func TestLeaderElectionWithKubernetesElector(t *testing.T) {
	eventBusAuthFileMountPath = "test"

	os.Setenv(common.EnvVarLeaderElection, "k8s")

	for _, config := range configs {
		elector, err := NewElector(context.TODO(), config, "", 0, "", "", "")
		assert.Nil(t, err)

		_, ok := elector.(*kubernetesElector)
		assert.True(t, ok)
	}
}
