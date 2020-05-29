package eventbus

import (
	"context"
	"fmt"
	"testing"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

const (
	testBusName   = "test-bus"
	testNATSImage = "test-image"
	testNamespace = "testNamespace"
	testUID       = "12341-asdf-2fees"
)

var (
	deleteTimestamp = metav1.Now().Rfc3339Copy()
)

func init() {
	v1alpha1.AddToScheme(scheme.Scheme)
	appv1.AddToScheme(scheme.Scheme)
	corev1.AddToScheme(scheme.Scheme)
}

func TestReconcile(t *testing.T) {
	tests := []struct {
		name    string
		initObj *v1alpha1.EventBus
		expect  *v1alpha1.EventBus
	}{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeClient := fake.NewFakeClient(test.initObj)
			r := &reconciler{
				client: fakeClient,
				scheme: scheme.Scheme,
				images: map[apicommon.EventBusType]string{
					apicommon.EventBusNATS: testNATSImage,
				},
				logger: ctrl.Log.WithName("test"),
			}
			err := r.reconcile(context.TODO(), test.initObj)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func fakeEventBus() *v1alpha1.EventBus {
	bus := &v1alpha1.EventBus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "EventBus",
		},
		ObjectMeta: fakeMetadata(testNamespace, testBusName),
		Spec: v1alpha1.EventBusSpec{
			NATS: &v1alpha1.NATSBus{
				Native: &v1alpha1.NativeStrategy{
					Size: 1,
				},
			},
		},
	}
	bus.ObjectMeta.SelfLink = ""
	return bus
}

func fakeMetadata(namespace, name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
		SelfLink:  fmt.Sprintf("/apis/argoproj.io/v1alpha1/namespaces/%s/eventbuses/%s", namespace, name),
		UID:       testUID,
	}
}

func getDeletingBusWithoutFinalizer() *v1alpha1.EventBus {
	bus := fakeEventBus()
	bus.DeletionTimestamp = &deleteTimestamp
	return bus
}

func getDeletingBus() *v1alpha1.EventBus {
	bus := getDeletingBusWithoutFinalizer()
	bus.Finalizers = []string{finalizerName}
	return bus
}

func getBusWithFinalizer() *v1alpha1.EventBus {
	bus := fakeEventBus()
	bus.Finalizers = []string{finalizerName}
	bus.Status.InitConditions()
	return bus
}

func getBusWithFinalizerAndServiceCreated() *v1alpha1.EventBus {
	bus := getBusWithFinalizer()
	bus.Status.MarkServiceCreated("Succeeded", "test")
	return bus
}

func getBusWithFinalizerAndServiceNotCreated() *v1alpha1.EventBus {
	bus := getBusWithFinalizer()
	bus.Status.MarkServiceNotCreated("Failed", "test")
	return bus
}

func getReadyBus() *v1alpha1.EventBus {
	bus := getBusWithFinalizerAndServiceCreated()
	bus.Status.MarkDeployed("Done", "test")
	bus.Status.MarkConfigured()
	return bus
}
