package eventbus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	testBusName        = "test-bus"
	testNATSImage      = "test-image"
	testStreamingImage = "test-steaming-image"
	testNamespace      = "testNamespace"
	testUID            = "12341-asdf-2fees"
	testURL            = "http://test"
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
					Size: 1,
					Auth: &v1alpha1.AuthStrategyToken,
					Persistence: &v1alpha1.PersistenceStrategy{
						Size: &volumeSize,
					},
				},
			},
		},
	}

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
					URL: testURL,
				},
			},
		},
	}
)

func init() {
	v1alpha1.AddToScheme(scheme.Scheme)
	appv1.AddToScheme(scheme.Scheme)
	corev1.AddToScheme(scheme.Scheme)
}

func TestReconcileNative(t *testing.T) {
	t.Run("native nats installation", func(t *testing.T) {
		obj := nativeBus.DeepCopyObject()
		testBus := obj.(*v1alpha1.EventBus)
		ctx := context.TODO()
		cl := fake.NewFakeClient(testBus)
		r := &reconciler{
			client:             cl,
			scheme:             scheme.Scheme,
			natsImage:          testNATSImage,
			natsStreamingImage: testStreamingImage,
			logger:             ctrl.Log.WithName("test"),
		}
		err := r.reconcile(ctx, testBus)
		assert.NoError(t, err)
		assert.True(t, testBus.Status.Status.IsReady())
		assert.NotNil(t, testBus.Status.Config.NATS)
		assert.NotEmpty(t, testBus.Status.Config.NATS.URL)
		assert.NotEmpty(t, testBus.Status.Config.NATS.ClusterID)
	})
}

func TestReconcileExotic(t *testing.T) {
	t.Run("native nats exotic", func(t *testing.T) {
		obj := exoticBus.DeepCopyObject()
		testBus := obj.(*v1alpha1.EventBus)
		ctx := context.TODO()
		cl := fake.NewFakeClient(testBus)
		r := &reconciler{
			client:             cl,
			scheme:             scheme.Scheme,
			natsImage:          testNATSImage,
			natsStreamingImage: testStreamingImage,
			logger:             ctrl.Log.WithName("test"),
		}
		err := r.reconcile(ctx, testBus)
		assert.NoError(t, err)
		assert.NotNil(t, testBus.Status.Config.NATS)
		assert.Equal(t, testURL, testBus.Status.Config.NATS.URL)
	})
}

func TestNeedsUpdate(t *testing.T) {
	t.Run("needs update", func(t *testing.T) {
		obj := nativeBus.DeepCopyObject()
		testBus := obj.(*v1alpha1.EventBus)
		cl := fake.NewFakeClient(testBus)
		r := &reconciler{
			client:             cl,
			scheme:             scheme.Scheme,
			natsImage:          testNATSImage,
			natsStreamingImage: testStreamingImage,
			logger:             ctrl.Log.WithName("test"),
		}
		assert.False(t, r.needsUpdate(nativeBus, testBus))
		r.addFinalizer(testBus)
		assert.True(t, contains(testBus.Finalizers, finalizerName))
		assert.True(t, r.needsUpdate(nativeBus, testBus))
		r.removeFinalizer(testBus)
		assert.False(t, contains(testBus.Finalizers, finalizerName))
		assert.False(t, r.needsUpdate(nativeBus, testBus))
		testBus.Status.MarkConfigured()
		assert.True(t, r.needsUpdate(nativeBus, testBus))
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
