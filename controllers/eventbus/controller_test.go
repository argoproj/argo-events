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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/argoproj/argo-events/controllers"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
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

	fakeConfig = &controllers.GlobalConfig{
		EventBus: &controllers.EventBusConfig{
			NATS: &controllers.StanConfig{
				Versions: []controllers.StanVersion{
					{
						Version:              "0.22.1",
						NATSStreamingImage:   "test-n-s-image",
						MetricsExporterImage: "test-n-s-m-image",
					},
				},
			},
			JetStream: &controllers.JetStreamConfig{
				Versions: []controllers.JetStreamVersion{
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
		r := &reconciler{
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
		r := &reconciler{
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

func TestNeedsUpdate(t *testing.T) {
	t.Run("needs update", func(t *testing.T) {
		testBus := nativeBus.DeepCopy()
		cl := fake.NewClientBuilder().Build()
		r := &reconciler{
			client:     cl,
			kubeClient: k8sfake.NewSimpleClientset(),
			scheme:     scheme.Scheme,
			config:     fakeConfig,
			logger:     zaptest.NewLogger(t).Sugar(),
		}
		assert.False(t, r.needsUpdate(nativeBus, testBus))
		controllerutil.AddFinalizer(testBus, finalizerName)
		assert.True(t, contains(testBus.Finalizers, finalizerName))
		assert.True(t, r.needsUpdate(nativeBus, testBus))
		controllerutil.RemoveFinalizer(testBus, finalizerName)
		assert.False(t, contains(testBus.Finalizers, finalizerName))
		assert.False(t, r.needsUpdate(nativeBus, testBus))
		testBus.Status.MarkConfigured()
		assert.False(t, r.needsUpdate(nativeBus, testBus))
	})
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
