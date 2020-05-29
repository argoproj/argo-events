package eventbus

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	testBusName   = "test-bus"
	testNATSImage = "test-image"
	testNamespace = "testNamespace"
	testUID       = "12341-asdf-2fees"
	testURL       = "http://test"
)

var (
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
			client: cl,
			scheme: scheme.Scheme,
			images: map[apicommon.EventBusType]string{
				apicommon.EventBusNATS: testNATSImage,
			},
			logger: ctrl.Log.WithName("test"),
		}
		err := r.reconcile(ctx, testBus)
		assert.NoError(t, err)
		svcList := &corev1.ServiceList{}
		err = cl.List(ctx, svcList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(svcList.Items))
		svc := svcList.Items[0]
		assert.Equal(t, fmt.Sprintf("eventbus-%s-svc", testBusName), svc.Name)
		ssList := &appv1.StatefulSetList{}
		err = cl.List(ctx, ssList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(ssList.Items))
		ss := ssList.Items[0]
		assert.Equal(t, fmt.Sprintf("eventbus-nats-%s", testBusName), ss.Name)
		assert.NotNil(t, testBus.Status.Config.NATS)
		assert.NotEmpty(t, testBus.Status.Config.NATS.URL)
	})
}

func TestReconcileExotic(t *testing.T) {
	t.Run("native nats exotic", func(t *testing.T) {
		obj := exoticBus.DeepCopyObject()
		testBus := obj.(*v1alpha1.EventBus)
		ctx := context.TODO()
		cl := fake.NewFakeClient(testBus)
		r := &reconciler{
			client: cl,
			scheme: scheme.Scheme,
			images: map[apicommon.EventBusType]string{
				apicommon.EventBusNATS: testNATSImage,
			},
			logger: ctrl.Log.WithName("test"),
		}
		err := r.reconcile(ctx, testBus)
		assert.NoError(t, err)
		assert.NotNil(t, testBus.Status.Config.NATS)
		assert.Equal(t, testURL, testBus.Status.Config.NATS.URL)
	})
}
