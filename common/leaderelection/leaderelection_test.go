package leaderelection

import (
	"context"
	"os"
	"testing"

	"github.com/argoproj/argo-events/common"
	dfv1 "github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

var (
	configs = []dfv1.BusConfig{
		{NATS: &dfv1.NATSConfig{}},
		{JetStream: &dfv1.JetStreamConfig{}},
		{JetStream: &dfv1.JetStreamConfig{AccessSecret: &v1.SecretKeySelector{}}},
	}
)

func TestLeaderElectionWithInvalidEventBus(t *testing.T) {
	elector, err := NewElector(context.TODO(), dfv1.BusConfig{}, "", 0, "", "", "")

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
