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
			Kind:       testEventBusKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testName,
		},
		Spec: v1alpha1.EventBusSpec{
			JetStream: &v1alpha1.JetStreamBus{
				Version: testVersion,
			},
		},
	}

	testJetStreamExoticBus = &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       testEventBusKind,
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
		for _, e := range []string{"POD_NAME", "SERVER_NAME", "POD_NAMESPACE", "CLUSTER_ADVERTISE", "JS_KEY_RAW", "JS_KEY"} {
			assert.Contains(t, envNames, e)
		}
		for _, e := range sts.Spec.Template.Spec.Containers[0].Env {
			if e.Name == "JS_KEY" {
				assert.Equal(t, `"$(JS_KEY_RAW)"`, e.Value)
			}
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
		// https://github.com/argoproj/argo-events/issues/4109
		// The JetStream encryption key must be at least 32 bytes so it
		// satisfies NATS' recommendation and the 112-bit HMAC minimum
		// required by OpenSSL FIPS providers.
		assert.GreaterOrEqual(t, len(s.Data[v1alpha1.JetStreamServerSecretEncryptionKey]), 32)
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
		conf := c.Data[v1alpha1.JetStreamConfigMapKey]
		assert.Contains(t, conf, "\n  key: \"$JS_KEY\"\n")
		assert.NotContains(t, conf, "prev_key")
	})

	t.Run("test create configmap with prev_key", func(t *testing.T) {
		testObj := testJetStreamEventBus.DeepCopy()
		testObj.Spec.JetStream.Version = testJetStreamPrevKeyVersion
		i.eventBus = testObj
		err := i.createConfigMap(ctx)
		assert.NoError(t, err)
		c := &corev1.ConfigMap{}
		err = cl.Get(ctx, types.NamespacedName{Namespace: testObj.Namespace, Name: generateJetStreamConfigMapName(testObj)}, c)
		assert.NoError(t, err)
		conf := c.Data[v1alpha1.JetStreamConfigMapKey]
		assert.Contains(t, conf, "\n  key: $JS_KEY\n")
		assert.Contains(t, conf, "\n  prev_key: \"$JS_KEY\"\n")
		assert.NotContains(t, conf, "\n  key: \"$JS_KEY\"\n")
	})
}

func TestJetStreamSupportsPrevKey(t *testing.T) {
	for version, want := range map[string]bool{
		"2.8.2":         false,
		"2.8.2-alpine":  false,
		"2.9.16":        false,
		"2.10.0":        true,
		"2.10.29":       true,
		"2.14.4":        true,
		"latest":        true,
		"":              false,
		"not-a-version": false,
	} {
		assert.Equal(t, want, jetStreamSupportsPrevKey(version), version)
	}
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
		st := testShortName
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
		assert.Equal(t, js.MetricsContainerTemplate.Resources, containers[metricsName].Resources)
		assert.Equal(t, js.ReloaderContainerTemplate.Resources, containers["reloader"].Resources)
	})

	t.Run("with probes", func(t *testing.T) {
		i.eventBus.Spec.JetStream = &v1alpha1.JetStreamBus{
			Version: testVersion,
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

func TestJetStreamUpgradeShortEncryptionKey(t *testing.T) {
	shortKey := []byte("abcdefghijkl") // 12 bytes, as generated before #4111

	tests := []struct {
		name        string
		version     string
		key         []byte
		sts         *appv1.StatefulSet
		wantUpgrade bool
	}{
		{name: "short key, store still on the literal key", version: "2.10.29", key: shortKey, wantUpgrade: true},
		{name: "short key, statefulset on the old configuration", version: "2.10.29", key: shortKey, sts: statefulSetWithEnv("JS_KEY"), wantUpgrade: true},
		{name: "short key, statefulset already reads the key from the secret", version: "2.10.29", key: shortKey, sts: statefulSetWithEnv("JS_KEY_RAW", "JS_KEY")},
		{name: "short key, server without prev_key support", version: "2.7.3", key: shortKey},
		{name: "key already long enough", version: "2.10.29", key: []byte("0123456789abcdef0123456789abcdef")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()
			eb := testJetStreamEventBus.DeepCopy()
			eb.UID = "eventbus-uid"
			eb.Spec.JetStream.Version = tc.version
			owner := []metav1.OwnerReference{*metav1.NewControllerRef(eb.GetObjectMeta(), v1alpha1.EventBusGroupVersionKind)}
			server := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: eb.Namespace, Name: generateJetStreamServerSecretName(eb), OwnerReferences: owner},
				Data:       map[string][]byte{v1alpha1.JetStreamServerSecretEncryptionKey: tc.key},
			}
			clientAuth := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Namespace: eb.Namespace, Name: generateJetStreamClientAuthSecretName(eb), OwnerReferences: owner},
				Data:       map[string][]byte{v1alpha1.JetStreamClientAuthSecretKey: []byte("username: u\npassword: p")},
			}
			builder := fake.NewClientBuilder().WithObjects(server.DeepCopy(), clientAuth.DeepCopy())
			if tc.sts != nil {
				sts := tc.sts.DeepCopy()
				sts.Namespace, sts.Name = eb.Namespace, generateJetStreamStatefulSetName(eb)
				builder = builder.WithObjects(sts)
			}
			cl := builder.Build()
			i := &jetStreamInstaller{
				client:     cl,
				kubeClient: k8sfake.NewSimpleClientset(server, clientAuth),
				eventBus:   eb,
				config:     fakeConfig,
				labels:     testLabels,
				logger:     zaptest.NewLogger(t).Sugar(),
			}

			assert.NoError(t, i.createSecrets(ctx))

			got := &corev1.Secret{}
			assert.NoError(t, cl.Get(ctx, types.NamespacedName{Namespace: eb.Namespace, Name: server.Name}, got))
			key := got.Data[v1alpha1.JetStreamServerSecretEncryptionKey]
			if tc.wantUpgrade {
				assert.Len(t, key, minEncryptionKeyLength)
				assert.NotEqual(t, tc.key, key)
			} else {
				assert.Equal(t, tc.key, key)
			}
		})
	}
}

func statefulSetWithEnv(names ...string) *appv1.StatefulSet {
	var env []corev1.EnvVar
	for _, n := range names {
		env = append(env, corev1.EnvVar{Name: n})
	}
	return &appv1.StatefulSet{
		Spec: appv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Env: env}}},
			},
		},
	}
}
