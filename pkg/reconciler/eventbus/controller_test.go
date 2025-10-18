package eventbus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/reconciler"
)

const (
	testBusName   = "test-bus"
	testNamespace = "testNamespace"
	testURL       = "http://test"
)

var (
	volumeSize = apiresource.MustParse("5Gi")

	nativeBus = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testBusName,
		},
		Spec: v1alpha1.EventBusSpec{
			NATS: &v1alpha1.NATSBus{
				Native: &v1alpha1.NativeStrategy{
					Replicas: 1,
					Auth:     &v1alpha1.AuthStrategyToken,
					Persistence: &v1alpha1.PersistenceStrategy{
						VolumeSize: &volumeSize,
					},
				},
			},
		},
	}

	cID = "test-cluster-id"

	exoticBus = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testBusName,
		},
		Spec: v1alpha1.EventBusSpec{
			NATS: &v1alpha1.NATSBus{
				Exotic: &v1alpha1.NATSConfig{
					ClusterID: &cID,
					URL:       testURL,
				},
			},
		},
	}

	fakeConfig = &reconciler.GlobalConfig{
		EventBus: &reconciler.EventBusConfig{
			NATS: &reconciler.StanConfig{
				Versions: []reconciler.StanVersion{
					{
						Version:              "0.22.1",
						NATSStreamingImage:   "test-n-s-image",
						MetricsExporterImage: "test-n-s-m-image",
					},
				},
			},
			JetStream: &reconciler.JetStreamConfig{
				Versions: []reconciler.JetStreamVersion{
					{
						Version:              "testVersion",
						NatsImage:            "testJSImage",
						ConfigReloaderImage:  "test-nats-rl-image",
						MetricsExporterImage: "testJSMetricsImage",
					},
				},
			},
		},
	}
)

func init() {
	_ = v1alpha1.AddToScheme(scheme.Scheme)
	_ = appv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)
}

func TestReconcileNative(t *testing.T) {
	t.Run("native nats installation", func(t *testing.T) {
		testBus := nativeBus.DeepCopy()
		ctx := context.TODO()
		cl := fake.NewClientBuilder().Build()
		r := &eventBusReconciler{
			client:     cl,
			kubeClient: k8sfake.NewSimpleClientset(),
			scheme:     scheme.Scheme,
			config:     fakeConfig,
			logger:     zaptest.NewLogger(t).Sugar(),
		}
		err := r.reconcile(ctx, testBus)
		assert.NoError(t, err)
		assert.True(t, testBus.Status.IsReady())
		assert.NotNil(t, testBus.Status.Config.NATS)
		assert.NotEmpty(t, testBus.Status.Config.NATS.URL)
		assert.NotEmpty(t, testBus.Status.Config.NATS.ClusterID)
	})
}

func TestReconcileExotic(t *testing.T) {
	t.Run("native nats exotic", func(t *testing.T) {
		testBus := exoticBus.DeepCopy()
		ctx := context.TODO()
		cl := fake.NewClientBuilder().Build()
		r := &eventBusReconciler{
			client:     cl,
			kubeClient: k8sfake.NewSimpleClientset(),
			scheme:     scheme.Scheme,
			config:     fakeConfig,
			logger:     zaptest.NewLogger(t).Sugar(),
		}
		err := r.reconcile(ctx, testBus)
		assert.NoError(t, err)
		assert.NotNil(t, testBus.Status.Config.NATS)
		assert.Equal(t, testURL, testBus.Status.Config.NATS.URL)
	})
}
