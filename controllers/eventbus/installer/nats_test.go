package installer

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	controllerscommon "github.com/argoproj/argo-events/controllers/common"
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
)

func TestMakeService(t *testing.T) {
	t.Run("make service spec", func(t *testing.T) {
		installer := getInstaller()
		got, err := installer.makeService()
		if err != nil {
			t.Error(err)
		}
		want := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("eventbus-%s-svc", testName),
				Namespace: testNamespace,
				Labels:    testLabels,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{Name: "client", Port: clientPort},
					{Name: "cluster", Port: clusterPort},
					{Name: "monitor", Port: monitorPort},
				},
				Type:     corev1.ServiceTypeClusterIP,
				Selector: testLabels,
			},
		}
		if err := controllerscommon.SetObjectMeta(testEventBus, want, v1alpha1.SchemaGroupVersionKind); err != nil {
			t.Error(err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("unexpected spec (-want, +got) = %v", diff)
		}
	})
}

func TestMakeStatefulSet(t *testing.T) {
	t.Run("make statefulset spec", func(t *testing.T) {
		installer := getInstaller()
		got, err := installer.makeStatefulSet()
		if err != nil {
			t.Error(err)
		}
		size := int32(1)
		want := &appv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      fmt.Sprintf("eventbus-nats-%s", testName),
				Labels:    testLabels,
			},
			Spec: appv1.StatefulSetSpec{
				Replicas:    &size,
				ServiceName: fmt.Sprintf("eventbus-%s-svc", testName),
				Selector: &metav1.LabelSelector{
					MatchLabels: testLabels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: testLabels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "nats",
								Image: testImage,
								Ports: []corev1.ContainerPort{
									{Name: "client", ContainerPort: clientPort},
									{Name: "cluster", ContainerPort: clusterPort},
									{Name: "monitor", ContainerPort: monitorPort},
								},
								LivenessProbe: &corev1.Probe{
									Handler: corev1.Handler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/",
											Port: intstr.FromInt(int(monitorPort)),
										},
									},
									InitialDelaySeconds: 10,
									TimeoutSeconds:      5,
								},
							},
						},
					},
				},
			},
		}
		if err := controllerscommon.SetObjectMeta(testEventBus, want, v1alpha1.SchemaGroupVersionKind); err != nil {
			t.Error(err)
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("unexpected spec (-want, +got) = %v", diff)
		}
	})
}

func getInstaller() *natsInstaller {
	v1alpha1.AddToScheme(scheme.Scheme)
	return &natsInstaller{
		client:   fake.NewFakeClient(testEventBus),
		eventBus: testEventBus,
		image:    testImage,
		labels:   testLabels,
		logger:   ctrl.Log.WithName("test"),
	}
}
