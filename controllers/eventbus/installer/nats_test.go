package installer

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	testNamespace = "test-ns"
	testName      = "test-name"
)

var (
	testLabels = map[string]string{"controller": "test-controller"}

	testNatsEventBus = &v1alpha1.EventBus{
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
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "test",
						},
					},
					ServiceAccountName: "test",
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
			client:   fake.NewClientBuilder().Build(),
			eventBus: testEventBusBad,
			config:   fakeConfig,
			labels:   testLabels,
			logger:   logging.NewArgoEventsLogger(),
		}
		_, err := installer.Install(context.TODO())
		assert.Error(t, err)
	})
}

func TestInstallationAuthtoken(t *testing.T) {
	kubeClient := k8sfake.NewSimpleClientset()
	t.Run("auth token installation", func(t *testing.T) {
		cl := fake.NewClientBuilder().Build()
		installer := NewNATSInstaller(cl, testNatsEventBus, fakeConfig, testLabels, kubeClient, zaptest.NewLogger(t).Sugar())
		busconf, err := installer.Install(context.TODO())
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
		for _, s := range svcList.Items {
			assert.True(t, strings.Contains(s.Name, "stan") || strings.Contains(s.Name, "metrics"))
		}

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
	kubeClient := k8sfake.NewSimpleClientset()
	t.Run("auth none installation", func(t *testing.T) {
		cl := fake.NewClientBuilder().Build()
		installer := NewNATSInstaller(cl, testEventBusAuthNone, fakeConfig, testLabels, kubeClient, zaptest.NewLogger(t).Sugar())
		busconf, err := installer.Install(context.TODO())
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
		cl := fake.NewClientBuilder().Build()
		installer := &natsInstaller{
			client:   cl,
			eventBus: testEventBusPersist,
			config:   fakeConfig,
			labels:   testLabels,
			logger:   logging.NewArgoEventsLogger(),
		}
		ss, err := installer.buildStatefulSet("svcName", "cmName", "secretName")
		assert.NoError(t, err)
		assert.True(t, len(ss.Spec.VolumeClaimTemplates) > 0)
	})

	t.Run("installation with image pull secrets", func(t *testing.T) {
		cl := fake.NewClientBuilder().Build()
		installer := &natsInstaller{
			client:   cl,
			eventBus: testNatsEventBus,
			config:   fakeConfig,
			labels:   testLabels,
			logger:   logging.NewArgoEventsLogger(),
		}
		ss, err := installer.buildStatefulSet("svcName", "cmName", "secretName")
		assert.NoError(t, err)
		assert.True(t, len(ss.Spec.Template.Spec.ImagePullSecrets) > 0)
	})

	t.Run("installation with priority class", func(t *testing.T) {
		cl := fake.NewClientBuilder().Build()
		eb := testNatsEventBus.DeepCopy()
		eb.Spec.NATS.Native.PriorityClassName = "test-class"
		installer := &natsInstaller{
			client:   cl,
			eventBus: eb,
			config:   fakeConfig,
			labels:   testLabels,
			logger:   logging.NewArgoEventsLogger(),
		}
		ss, err := installer.buildStatefulSet("svcName", "cmName", "secretName")
		assert.NoError(t, err)
		assert.Equal(t, ss.Spec.Template.Spec.PriorityClassName, "test-class")
	})
}

func TestBuildServiceAccountStatefulSetSpec(t *testing.T) {
	t.Run("installation with Service Account Name", func(t *testing.T) {
		cl := fake.NewClientBuilder().Build()
		installer := &natsInstaller{
			client:   cl,
			eventBus: testNatsEventBus,
			config:   fakeConfig,
			labels:   testLabels,
			logger:   logging.NewArgoEventsLogger(),
		}
		ss, err := installer.buildStatefulSet("svcName", "cmName", "secretName")
		assert.NoError(t, err)
		assert.True(t, len(ss.Spec.Template.Spec.ServiceAccountName) > 0)
	})
}

func TestBuildConfigMap(t *testing.T) {
	t.Run("test build config map", func(t *testing.T) {
		cl := fake.NewClientBuilder().Build()
		installer := &natsInstaller{
			client:   cl,
			eventBus: testNatsEventBus,
			config:   fakeConfig,
			labels:   testLabels,
			logger:   logging.NewArgoEventsLogger(),
		}
		cm, err := installer.buildConfigMap()
		assert.NoError(t, err)
		assert.NotNil(t, cm)
		conf, ok := cm.Data[configMapKey]
		assert.True(t, ok)
		assert.True(t, strings.Contains(conf, "routes:"))
		svcName := generateServiceName(testNatsEventBus)
		ssName := generateStatefulSetName(testNatsEventBus)
		r := fmt.Sprintf("nats://%s-%s.%s.%s.svc:%s", ssName, "0", svcName, testNamespace, strconv.Itoa(int(clusterPort)))
		lines := strings.Split(conf, `\n`)
		for _, l := range lines {
			l = strings.Trim(l, " ")
			if strings.HasPrefix(l, "routes:") {
				assert.True(t, strings.Contains(l, r))
			}
		}
	})
}
