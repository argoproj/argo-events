package common

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"
)

func getInformerFactory() *ArgoEventInformerFactory {
	clientset := fake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(clientset, 0)
	ownerInformer := informerFactory.Core().V1().Pods().Informer()
	return &ArgoEventInformerFactory{
		OwnerKind:             "foo",
		OwnerInformer:         ownerInformer,
		SharedInformerFactory: informerFactory,
		Queue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

func TestInformer(t *testing.T) {
	convey.Convey("Given a gateway controller", t, func() {
		factory := getInformerFactory()
		convey.Convey("Get a new gateway pod informer and make sure its not nil", func() {
			i := factory.NewPodInformer()
			convey.So(i, convey.ShouldNotBeNil)
		})
	})

	convey.Convey("Given a gateway controller", t, func() {
		factory := getInformerFactory()
		convey.Convey("Get a new gateway service informer and make sure its not nil", func() {
			i := factory.NewServiceInformer()
			convey.So(i, convey.ShouldNotBeNil)
		})
	})
}
