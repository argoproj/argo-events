package installer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	testExoticKafkaName = "test-kafka"
	testExoticKafkaURL  = "kafka:9092"
)

var (
	testKafkaExoticBus = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testExoticKafkaName,
		},
		Spec: v1alpha1.EventBusSpec{
			Kafka: &v1alpha1.KafkaBus{
				Exotic: &v1alpha1.KafkaConfig{
					URL: testExoticKafkaURL,
				},
			},
		},
	}
)

func TestInstallationKafkaExotic(t *testing.T) {
	t.Run("installation with exotic kafka config", func(t *testing.T) {
		installer := NewExoticKafkaInstaller(testKafkaExoticBus, logging.NewArgoEventsLogger())
		conf, err := installer.Install(context.TODO())
		assert.NoError(t, err)
		assert.NotNil(t, conf.Kafka)
		assert.Equal(t, conf.Kafka.URL, testExoticKafkaURL)
	})
}

func TestUninstallationKafkaExotic(t *testing.T) {
	t.Run("uninstallation with exotic kafka config", func(t *testing.T) {
		installer := NewExoticKafkaInstaller(testKafkaExoticBus, logging.NewArgoEventsLogger())
		err := installer.Uninstall(context.TODO())
		assert.NoError(t, err)
	})
}
