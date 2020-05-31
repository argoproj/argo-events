package installer

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	testNamespace = "test-ns"
	testName      = "test-name"
	testImage     = "test-image"
)

var (
	testLabels   = map[string]string{"controller": "test-controller"}
	testEventBus = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: v1alpha1.EventBusSpec{
			NATS: &v1alpha1.NATSBus{
				Native: &v1alpha1.NativeStrategy{},
			},
		},
	}
	testEventBusBad = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: v1alpha1.EventBusSpec{},
	}
)

func init() {
	v1alpha1.AddToScheme(scheme.Scheme)
	appv1.AddToScheme(scheme.Scheme)
	corev1.AddToScheme(scheme.Scheme)
}

func TestBadInstallation(t *testing.T) {
	t.Run("bad installation", func(t *testing.T) {
		installer := &natsInstaller{
			client:   fake.NewFakeClient(testEventBusBad),
			eventBus: testEventBusBad,
			image:    testImage,
			labels:   testLabels,
			logger:   ctrl.Log.WithName("test"),
		}
		_, err := installer.Install()
		assert.Error(t, err)
	})
}

func TestGoodInstallation(t *testing.T) {
	t.Run("good installation", func(t *testing.T) {
		cl := fake.NewFakeClient(testEventBus)
		installer := &natsInstaller{
			client:   cl,
			eventBus: testEventBus,
			image:    testImage,
			labels:   testLabels,
			logger:   ctrl.Log.WithName("test"),
		}
		busconf, err := installer.Install()
		assert.NoError(t, err)
		assert.NotNil(t, busconf.NATS)
		assert.NotEmpty(t, busconf.NATS.URL)
		assert.Equal(t, busconf.NATS.Auth, v1alpha1.AuthStrategyToken)

		ctx := context.TODO()
		svcList := &corev1.ServiceList{}
		err = cl.List(ctx, svcList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(svcList.Items))
		svc := svcList.Items[0]
		assert.Equal(t, fmt.Sprintf("eventbus-%s-svc", testName), svc.Name)

		cmList := &corev1.ConfigMapList{}
		err = cl.List(ctx, cmList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cmList.Items))
		cm := cmList.Items[0]
		assert.Equal(t, fmt.Sprintf("eventbus-nats-%s-configmap", testName), cm.Name)

		secretList := &corev1.SecretList{}
		err = cl.List(ctx, secretList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, len(secretList.Items))
		for _, s := range secretList.Items {
			assert.True(t, strings.Contains(s.Name, "server") || strings.Contains(s.Name, "client"))
		}

		ssList := &appv1.StatefulSetList{}
		err = cl.List(ctx, ssList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(ssList.Items))
		ss := ssList.Items[0]
		assert.Equal(t, fmt.Sprintf("eventbus-nats-%s", testName), ss.Name)
	})
}
