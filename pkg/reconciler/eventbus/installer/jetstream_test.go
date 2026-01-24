package installer

import (
	"context"

	"testing"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	testJetStreamEventBus = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: v1alpha1.EventBusSpec{
			JetStream: &v1alpha1.JetStreamBus{
				Version: "2.7.3",
			},
		},
	}

	testJetStreamExoticBus = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: v1alpha1.EventBusSpec{
			JetStreamExotic: &v1alpha1.JetStreamConfig{
				URL: "nats://nats:4222",
			},
		},
	}
)

func TestJetStreamBadInstallation(t *testing.T) {
	t.Run("bad installation", func(t *testing.T) {
		badEventBus := testJetStreamEventBus.DeepCopy()
		badEventBus.Spec.JetStream = nil
		installer := &jetStreamInstaller{
			client:     fake.NewClientBuilder().Build(),
			kubeClient: k8sfake.NewSimpleClientset(),
			eventBus:   badEventBus,
			config:     fakeConfig,
			labels:     testLabels,
			logger:     zaptest.NewLogger(t).Sugar(),
		}
		_, err := installer.Install(context.TODO())
		assert.Error(t, err)
	})
}

func TestJetStreamGenerateNames(t *testing.T) {
	n := generateJetStreamStatefulSetName(testJetStreamEventBus)
	assert.Equal(t, "eventbus-"+testJetStreamEventBus.Name+"-js", n)
	n = generateJetStreamServerSecretName(testJetStreamEventBus)
	assert.Equal(t, "eventbus-"+testJetStreamEventBus.Name+"-js-server", n)
	n = generateJetStreamClientAuthSecretName(testJetStreamEventBus)
	assert.Equal(t, "eventbus-"+testJetStreamEventBus.Name+"-js-client-auth", n)
	n = generateJetStreamConfigMapName(testJetStreamEventBus)
	assert.Equal(t, "eventbus-"+testJetStreamEventBus.Name+"-js-config", n)
	n = generateJetStreamPVCName(testJetStreamEventBus)
	assert.Equal(t, "eventbus-"+testJetStreamEventBus.Name+"-js-vol", n)
	n = generateJetStreamServiceName(testJetStreamEventBus)
	assert.Equal(t, "eventbus-"+testJetStreamEventBus.Name+"-js-svc", n)
}

func TestJetStreamCreateObjects(t *testing.T) {
	cl := fake.NewClientBuilder().Build()
	ctx := context.TODO()
	i := &jetStreamInstaller{
		client:     cl,
		kubeClient: k8sfake.NewSimpleClientset(),
		eventBus:   testJetStreamEventBus,
		config:     fakeConfig,
		labels:     testLabels,
		logger:     zaptest.NewLogger(t).Sugar(),
	}

	t.Run("test create sts", func(t *testing.T) {
		testObj := testJetStreamEventBus.DeepCopy()
		i.eventBus = testObj
		err := i.createStatefulSet(ctx)
		assert.NoError(t, err)
		sts := &appv1.StatefulSet{}
		err = cl.Get(ctx, types.NamespacedName{Namespace: testObj.Namespace, Name: generateJetStreamStatefulSetName(testObj)}, sts)
		assert.NoError(t, err)
		assert.Equal(t, 3, len(sts.Spec.Template.Spec.Containers))
		assert.Contains(t, sts.Annotations, v1alpha1.AnnotationResourceSpecHash)
		assert.Equal(t, testJetStreamImage, sts.Spec.Template.Spec.Containers[0].Image)
		assert.Equal(t, testJSReloaderImage, sts.Spec.Template.Spec.Containers[1].Image)
		assert.Equal(t, testJetStreamExporterImage, sts.Spec.Template.Spec.Containers[2].Image)
		assert.True(t, len(sts.Spec.Template.Spec.Volumes) > 1)
		envNames := []string{}
		for _, e := range sts.Spec.Template.Spec.Containers[0].Env {
			envNames = append(envNames, e.Name)
		}
		for _, e := range []string{"POD_NAME", "SERVER_NAME", "POD_NAMESPACE", "CLUSTER_ADVERTISE", "JS_KEY"} {
			assert.Contains(t, envNames, e)
		}
	})

	t.Run("test create svc", func(t *testing.T) {
		testObj := testJetStreamEventBus.DeepCopy()
		i.eventBus = testObj
		err := i.createService(ctx)
		assert.NoError(t, err)
		svc := &corev1.Service{}
		err = cl.Get(ctx, types.NamespacedName{Namespace: testObj.Namespace, Name: generateJetStreamServiceName(testObj)}, svc)
		assert.NoError(t, err)
		assert.Equal(t, 4, len(svc.Spec.Ports))
		assert.Contains(t, svc.Annotations, v1alpha1.AnnotationResourceSpecHash)
	})

	t.Run("test create auth secrets", func(t *testing.T) {
		testObj := testJetStreamEventBus.DeepCopy()
		i.eventBus = testObj
		err := i.createSecrets(ctx)
		assert.NoError(t, err)
		s := &corev1.Secret{}
		err = cl.Get(ctx, types.NamespacedName{Namespace: testObj.Namespace, Name: generateJetStreamServerSecretName(testObj)}, s)
		assert.NoError(t, err)
		assert.Equal(t, 8, len(s.Data))
		assert.Contains(t, s.Data, v1alpha1.JetStreamServerSecretAuthKey)
		assert.Contains(t, s.Data, v1alpha1.JetStreamServerSecretEncryptionKey)
		assert.Contains(t, s.Data, v1alpha1.JetStreamServerPrivateKeyKey)
		assert.Contains(t, s.Data, v1alpha1.JetStreamServerCertKey)
		assert.Contains(t, s.Data, v1alpha1.JetStreamServerCACertKey)
		assert.Contains(t, s.Data, v1alpha1.JetStreamClusterPrivateKeyKey)
		assert.Contains(t, s.Data, v1alpha1.JetStreamClusterCertKey)
		assert.Contains(t, s.Data, v1alpha1.JetStreamClusterCACertKey)
		s = &corev1.Secret{}
		err = cl.Get(ctx, types.NamespacedName{Namespace: testObj.Namespace, Name: generateJetStreamClientAuthSecretName(testObj)}, s)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(s.Data))
		assert.Contains(t, s.Data, v1alpha1.JetStreamClientAuthSecretKey)
	})

	t.Run("test create configmap", func(t *testing.T) {
		testObj := testJetStreamEventBus.DeepCopy()
		i.eventBus = testObj
		err := i.createConfigMap(ctx)
		assert.NoError(t, err)
		c := &corev1.ConfigMap{}
		err = cl.Get(ctx, types.NamespacedName{Namespace: testObj.Namespace, Name: generateJetStreamConfigMapName(testObj)}, c)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(c.Data))
		assert.Contains(t, c.Annotations, v1alpha1.AnnotationResourceSpecHash)
	})
}

func TestBuildJetStreamStatefulSetSpec(t *testing.T) {
	cl := fake.NewClientBuilder().Build()
	i := &jetStreamInstaller{
		client:   cl,
		eventBus: testJetStreamEventBus,
		config:   fakeConfig,
		labels:   testLabels,
		logger:   zaptest.NewLogger(t).Sugar(),
	}

	t.Run("without persistence", func(t *testing.T) {
		s := i.buildStatefulSetSpec(&fakeConfig.EventBus.JetStream.Versions[0])
		assert.Equal(t, int32(3), *s.Replicas)
		assert.Equal(t, generateJetStreamServiceName(testJetStreamEventBus), s.ServiceName)
		assert.Equal(t, testJetStreamImage, s.Template.Spec.Containers[0].Image)
		assert.Equal(t, testJSReloaderImage, s.Template.Spec.Containers[1].Image)
		assert.Equal(t, testJetStreamExporterImage, s.Template.Spec.Containers[2].Image)
		assert.Equal(t, "test-controller", s.Selector.MatchLabels["controller"])
		assert.Equal(t, jsClientPort, s.Template.Spec.Containers[0].Ports[0].ContainerPort)
		assert.Equal(t, jsClusterPort, s.Template.Spec.Containers[0].Ports[1].ContainerPort)
		assert.Equal(t, jsMonitorPort, s.Template.Spec.Containers[0].Ports[2].ContainerPort)
		assert.Equal(t, jsMetricsPort, s.Template.Spec.Containers[2].Ports[0].ContainerPort)
		assert.False(t, len(s.VolumeClaimTemplates) > 0)
		assert.True(t, len(s.Template.Spec.Volumes) > 0)
	})

	t.Run("with persistence", func(t *testing.T) {
		st := "test"
		i.eventBus.Spec.JetStream = &v1alpha1.JetStreamBus{
			Persistence: &v1alpha1.PersistenceStrategy{
				StorageClassName: &st,
			},
		}
		s := i.buildStatefulSetSpec(&fakeConfig.EventBus.JetStream.Versions[0])
		assert.True(t, len(s.VolumeClaimTemplates) > 0)
	})

	t.Run("with resource requests", func(t *testing.T) {
		i.eventBus.Spec.JetStream = &v1alpha1.JetStreamBus{
			ContainerTemplate: &v1alpha1.ContainerTemplate{
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    apiresource.Quantity{Format: "1"},
						corev1.ResourceMemory: apiresource.Quantity{Format: "350Mi"},
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    apiresource.Quantity{Format: "1"},
						corev1.ResourceMemory: apiresource.Quantity{Format: "400Mi"},
					},
				},
			},

			MetricsContainerTemplate: &v1alpha1.ContainerTemplate{
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    apiresource.Quantity{Format: "1"},
						corev1.ResourceMemory: apiresource.Quantity{Format: "200Mi"},
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    apiresource.Quantity{Format: "1"},
						corev1.ResourceMemory: apiresource.Quantity{Format: "200Mi"},
					},
				},
			},

			ReloaderContainerTemplate: &v1alpha1.ContainerTemplate{
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    apiresource.Quantity{Format: ".3"},
						corev1.ResourceMemory: apiresource.Quantity{Format: "100Mi"},
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    apiresource.Quantity{Format: ".5"},
						corev1.ResourceMemory: apiresource.Quantity{Format: "100Mi"},
					},
				},
			},
		}

		statefulSpec := i.buildStatefulSetSpec(&fakeConfig.EventBus.JetStream.Versions[0])

		podContainers := statefulSpec.Template.Spec.Containers
		containers := map[string]*corev1.Container{}
		for idx := range podContainers {
			containers[podContainers[idx].Name] = &podContainers[idx]
		}

		js := i.eventBus.Spec.JetStream
		assert.Equal(t, js.ContainerTemplate.Resources, containers["main"].Resources)
		assert.Equal(t, js.MetricsContainerTemplate.Resources, containers["metrics"].Resources)
		assert.Equal(t, js.ReloaderContainerTemplate.Resources, containers["reloader"].Resources)
	})

	t.Run("with probes", func(t *testing.T) {
		i.eventBus.Spec.JetStream = &v1alpha1.JetStreamBus{
			Version: "2.7.3",
		}
		s := i.buildStatefulSetSpec(&fakeConfig.EventBus.JetStream.Versions[0])
		mainContainer := s.Template.Spec.Containers[0]

		// Verify StartupProbe
		assert.NotNil(t, mainContainer.StartupProbe)
		assert.NotNil(t, mainContainer.StartupProbe.HTTPGet)
		assert.Equal(t, "/healthz", mainContainer.StartupProbe.HTTPGet.Path)
		assert.Equal(t, int32(30), mainContainer.StartupProbe.FailureThreshold)

		// Verify LivenessProbe
		assert.NotNil(t, mainContainer.LivenessProbe)
		assert.NotNil(t, mainContainer.LivenessProbe.HTTPGet)
		assert.Equal(t, "/", mainContainer.LivenessProbe.HTTPGet.Path)
		assert.Equal(t, int32(10), mainContainer.LivenessProbe.InitialDelaySeconds)
		assert.Equal(t, int32(30), mainContainer.LivenessProbe.PeriodSeconds)

		// Verify ReadinessProbe
		assert.NotNil(t, mainContainer.ReadinessProbe)
		assert.NotNil(t, mainContainer.ReadinessProbe.HTTPGet)
		assert.Equal(t, "/healthz", mainContainer.ReadinessProbe.HTTPGet.Path)
		assert.Equal(t, int32(10), mainContainer.ReadinessProbe.InitialDelaySeconds)
		assert.Equal(t, int32(5), mainContainer.ReadinessProbe.PeriodSeconds)
		assert.Equal(t, int32(5), mainContainer.ReadinessProbe.TimeoutSeconds)
	})
}

func TestJetStreamGetServiceSpec(t *testing.T) {
	cl := fake.NewClientBuilder().Build()
	i := &jetStreamInstaller{
		client:   cl,
		eventBus: testJetStreamEventBus,
		config:   fakeConfig,
		labels:   testLabels,
		logger:   zaptest.NewLogger(t).Sugar(),
	}
	spec := i.buildJetStreamServiceSpec()
	assert.Equal(t, 4, len(spec.Ports))
	assert.Equal(t, corev1.ClusterIPNone, spec.ClusterIP)
}

func Test_JSBufferGetReplicas(t *testing.T) {
	s := v1alpha1.JetStreamBus{}
	assert.Equal(t, 3, s.GetReplicas())
	five := int32(5)
	s.Replicas = &five
	assert.Equal(t, 5, s.GetReplicas())
}
