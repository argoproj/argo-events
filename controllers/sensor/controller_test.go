package sensor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	eventbusv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
)

const (
	testImage = "test-image"
)

var (
	SensorControllerConfigmap = "sensor-controller-configmap"

	SensorControllerInstanceID = "argo-events"
)

func TestReconcile(t *testing.T) {
	t.Run("test reconcile without eventbus", func(t *testing.T) {
		ctx := context.TODO()
		cl := fake.NewFakeClient(sensorObj)
		r := &reconciler{
			client:      cl,
			scheme:      scheme.Scheme,
			sensorImage: testImage,
			logger:      ctrl.Log.WithName("test"),
		}
		err := r.reconcile(ctx, sensorObj)
		assert.Error(t, err)
		assert.False(t, sensorObj.Status.IsReady())
	})

	t.Run("test reconcile with eventbus", func(t *testing.T) {
		ctx := context.TODO()
		cl := fake.NewFakeClient(sensorObj)
		testBus := fakeEventBus.DeepCopy()
		testBus.Status.MarkDeployed("test", "test")
		testBus.Status.MarkConfigured()
		err := cl.Create(ctx, testBus)
		assert.Nil(t, err)
		r := &reconciler{
			client:      cl,
			scheme:      scheme.Scheme,
			sensorImage: testImage,
			logger:      ctrl.Log.WithName("test"),
		}
		err = r.reconcile(ctx, sensorObj)
		assert.NoError(t, err)
		assert.True(t, sensorObj.Status.IsReady())
	})
}

func init() {
	_ = eventbusv1alpha1.AddToScheme(scheme.Scheme)
	_ = v1alpha1.AddToScheme(scheme.Scheme)
	_ = appv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)
}
