package eventbus

import (
	"context"
	"sync"
	"testing"

	fuzz "github.com/AdaLogics/go-fuzz-headers"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/argoproj/argo-events/controllers"

	"github.com/argoproj/argo-events/common/logging"
	"github.com/argoproj/argo-events/pkg/apis/eventbus/v1alpha1"
)

var initter sync.Once

func initScheme() {
	_ = v1alpha1.AddToScheme(scheme.Scheme)
	_ = appv1.AddToScheme(scheme.Scheme)
	_ = corev1.AddToScheme(scheme.Scheme)
}

func FuzzEventbusReconciler(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		initter.Do(initScheme)
		f := fuzz.NewConsumer(data)
		nativeBus := &v1alpha1.EventBus{}
		err := f.GenerateStruct(nativeBus)
		if err != nil {
			return
		}
		cl := fake.NewClientBuilder().Build()
		config := &controllers.GlobalConfig{}
		err = f.GenerateStruct(config)
		if err != nil {
			return
		}
		r := &reconciler{
			client: cl,
			scheme: scheme.Scheme,
			config: config,
			logger: logging.NewArgoEventsLogger(),
		}
		ctx := context.Background()
		_ = r.reconcile(ctx, nativeBus)
		_ = r.needsUpdate(nativeBus, nativeBus)
	})
}
