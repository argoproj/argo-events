package gateway

import (
	"fmt"
	fakegateway "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"
	"testing"
)

func getGatewayController() *GatewayController {
	return &GatewayController{
		ConfigMap: "configmap",
		Namespace: "testing",
		Config: GatewayControllerConfig{
			Namespace: "testing",
		},
		kubeClientset:    fake.NewSimpleClientset(),
		gatewayClientset: fakegateway.NewSimpleClientset(),
		queue:            workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

func TestProcessNextItem(t *testing.T) {
	controller := getController()
	controller.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	controller.informer = controller.newGatewayInformer()

	controller.queue.Add("hi")
	res := controller.processNextItem()
	assert.True(t, res)

	controller.queue.ShutDown()
	res = controller.processNextItem()
	assert.False(t, res)
}

func TestHandleErr(t *testing.T) {
	controller := getGatewayController()
	controller.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	controller.queue.Add("hi")
	controller.handleErr(nil, "hi")

	controller.queue.Add("bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
	controller.handleErr(fmt.Errorf("real error"), "bye")
}
