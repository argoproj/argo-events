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
	testNamespace      = "test-ns"
	testName           = "test-name"
	testStreamingImage = "test-s-image"
)

var (
	testLabels = map[string]string{"controller": "test-controller"}

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
				Native: &v1alpha1.NativeStrategy{
					Auth: &v1alpha1.AuthStrategyToken,
				},
			},
		},
	}

	testEventBusPersist = &v1alpha1.EventBus{
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
				Native: &v1alpha1.NativeStrategy{
					Persistence: &v1alpha1.PersistenceStrategy{},
				},
			},
		},
	}

	testEventBusAuthNone = &v1alpha1.EventBus{
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
				Native: &v1alpha1.NativeStrategy{
					Auth: &v1alpha1.AuthStrategyNone,
				},
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
	_ = v1alpha1.AddToScheme(scheme.Scheme)
	_ = appv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)
}

func TestBadInstallation(t *testing.T) {
	t.Run("bad installation", func(t *testing.T) {
		installer := &natsInstaller{
			client:         fake.NewFakeClient(testEventBusBad),
			eventBus:       testEventBusBad,
			streamingImage: testStreamingImage,
			labels:         testLabels,
			logger:         ctrl.Log.WithName("test"),
		}
		_, err := installer.Install()
		assert.Error(t, err)
	})
}

func TestInstallationAuthtoken(t *testing.T) {
	t.Run("auth token installation", func(t *testing.T) {
		cl := fake.NewFakeClient(testEventBus)
		installer := NewNATSInstaller(cl, testEventBus, testStreamingImage, testLabels, ctrl.Log.WithName("test"))
		busconf, err := installer.Install()
		assert.NoError(t, err)
		assert.NotNil(t, busconf.NATS)
		assert.NotEmpty(t, busconf.NATS.URL)
		assert.Equal(t, busconf.NATS.Auth, &v1alpha1.AuthStrategyToken)

		ctx := context.TODO()
		svcList := &corev1.ServiceList{}
		err = cl.List(ctx, svcList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(svcList.Items))
		assert.Equal(t, svcList.Items[0].Name, fmt.Sprintf("eventbus-%s-stan-svc", testName))

		cmList := &corev1.ConfigMapList{}
		err = cl.List(ctx, cmList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cmList.Items))
		assert.Equal(t, cmList.Items[0].Name, fmt.Sprintf("eventbus-%s-stan-configmap", testName))

		ssList := &appv1.StatefulSetList{}
		err = cl.List(ctx, ssList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(ssList.Items))
		assert.Equal(t, ssList.Items[0].Name, fmt.Sprintf("eventbus-%s-stan", testName))

		secretList := &corev1.SecretList{}
		err = cl.List(ctx, secretList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, len(secretList.Items))
		for _, s := range secretList.Items {
			assert.True(t, strings.Contains(s.Name, "server") || strings.Contains(s.Name, "client"))
		}
	})
}

func TestInstallationAuthNone(t *testing.T) {
	t.Run("auth none installation", func(t *testing.T) {
		cl := fake.NewFakeClient(testEventBusAuthNone)
		installer := NewNATSInstaller(cl, testEventBusAuthNone, testStreamingImage, testLabels, ctrl.Log.WithName("test"))
		busconf, err := installer.Install()
		assert.NoError(t, err)
		assert.NotNil(t, busconf.NATS)
		assert.NotEmpty(t, busconf.NATS.URL)
		assert.Equal(t, busconf.NATS.Auth, &v1alpha1.AuthStrategyNone)

		ctx := context.TODO()
		svcList := &corev1.ServiceList{}
		err = cl.List(ctx, svcList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(svcList.Items))

		cmList := &corev1.ConfigMapList{}
		err = cl.List(ctx, cmList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(cmList.Items))

		secretList := &corev1.SecretList{}
		err = cl.List(ctx, secretList, &client.ListOptions{
			Namespace: testNamespace,
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(secretList.Items))
		assert.True(t, strings.Contains(secretList.Items[0].Name, "server"))
		assert.True(t, len(secretList.Items[0].Data[serverAuthSecretKey]) == 0)
	})
}

func TestBuildPersistStatefulSetSpec(t *testing.T) {
	t.Run("installation with persistence", func(t *testing.T) {
		cl := fake.NewFakeClient(testEventBusPersist)
		installer := &natsInstaller{
			client:         cl,
			eventBus:       testEventBusPersist,
			streamingImage: testStreamingImage,
			labels:         testLabels,
			logger:         ctrl.Log.WithName("test"),
		}
		ss, err := installer.buildStatefulSet("svcName", "cmName", "secretName")
		assert.NoError(t, err)
		assert.True(t, len(ss.Spec.VolumeClaimTemplates) > 0)
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
