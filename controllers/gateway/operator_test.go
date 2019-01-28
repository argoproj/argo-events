/*
Copyright 2018 BlackRock, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gateway

import (
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testGatewayStr = `apiVersion: argoproj.io/v1alpha1
kind: Gateway
metadata:
 name: webhook-gateway
 namespace: argo-events
 labels:
   gateways.argoproj.io/gateway-controller-instanceid: argo-events
   gateway-name: "webhook-gateway"
spec:
  configMap: "webhook-gateway-configmap"
  type: "webhook"
  processorPort: "9330"
  eventProtocol:
    type: "NATS"
    nats:
      url: "nats://nats.argo-events:4222"
      type: "Standard"
  eventVersion: "1.0"
  deploySpec:
    metadata:
      name: "webhook-gateway"
      labels:
        gateway-name: "webhook-gateway"
    spec:
      containers:
        - name: "gateway-client"
          image: "argoproj/gateway-client:v0.6.2"
          imagePullPolicy: "Always"
          command: ["/bin/gateway-client"]
        - name: "webhook-events"
          image: "argoproj/webhook-gateway:v0.6.2"
          imagePullPolicy: "Always"
          command: ["/bin/webhook-gateway"]
      serviceAccountName: "argo-events-sa"
  serviceSpec:
    metadata:
      name: webhook-gateway-svc
    spec:
      selector:
        gateway-name: "webhook-gateway"
      ports:
        - port: 12000
          targetPort: 12000
      type: LoadBalancer`

func getGateway() (*v1alpha1.Gateway, error) {
	gwBytes := []byte(testGatewayStr)
	var gateway v1alpha1.Gateway
	err := yaml.Unmarshal(gwBytes, &gateway)
	return &gateway, err
}

func TestGatewayOperateLifecycle(t *testing.T) {
	convey.Convey("Given a gateway resource spec, parse it", t, func() {
		fakeController := getGatewayController()
		gateway, err := getGateway()
		convey.Convey("Make sure no error occurs", func() {
			convey.So(err, convey.ShouldBeNil)

			convey.Convey("Create the gateway", func() {
				gateway, err = fakeController.gatewayClientset.ArgoprojV1alpha1().Gateways(fakeController.Config.Namespace).Create(gateway)

				convey.Convey("No error should occur and gateway resource should not be empty", func() {
					convey.So(err, convey.ShouldBeNil)
					convey.So(gateway, convey.ShouldNotBeNil)

					convey.Convey("Create a new gateway operation context", func() {
						goc := newGatewayOperationCtx(gateway, fakeController)
						convey.So(goc, convey.ShouldNotBeNil)

						convey.Convey("Operate on new gateway", func() {
							err := goc.operate()

							convey.Convey("Operation must succeed", func() {
								convey.So(err, convey.ShouldBeNil)

								convey.Convey("A gateway pod and service must be created", func() {
									pod, err := goc.controller.kubeClientset.CoreV1().Pods(gateway.Namespace).Get(gateway.Name, metav1.GetOptions{})
									convey.So(err, convey.ShouldBeNil)
									convey.So(pod, convey.ShouldNotBeNil)

									svc, err := goc.controller.kubeClientset.CoreV1().Services(gateway.Namespace).Get("webhook-gateway-svc", metav1.GetOptions{})
									convey.So(err, convey.ShouldBeNil)
									convey.So(svc, convey.ShouldNotBeNil)
								})
							})
						})

						convey.Convey("Operate on gateway in running state", func() {
							err := goc.operate()
							convey.So(err, convey.ShouldBeNil)
						})

						convey.Convey("Mark gateway state as error and operate", func() {
							goc.markGatewayPhase(v1alpha1.NodePhaseError, "gateway is in error state")
							err := goc.operate()
							convey.So(err, convey.ShouldBeNil)
							gateway, err := goc.controller.gatewayClientset.ArgoprojV1alpha1().Gateways(gateway.Namespace).Get(gateway.Name, metav1.GetOptions{})
							convey.So(err, convey.ShouldBeNil)
							convey.So(gateway.Status.Phase, convey.ShouldEqual, v1alpha1.NodePhaseError)
						})

						convey.Convey("Delete gateway and make sure both pod and service gets deleted", func() {
							err := fakeController.gatewayClientset.ArgoprojV1alpha1().Gateways(gateway.Namespace).Delete(gateway.Name, &metav1.DeleteOptions{})
							convey.So(err, convey.ShouldBeNil)

							gatewayPod, err := fakeController.kubeClientset.CoreV1().Pods(gateway.Namespace).Get(gateway.Name, metav1.GetOptions{})
							convey.So(err, convey.ShouldNotBeNil)
							convey.So(gatewayPod, convey.ShouldBeNil)

							gatewaySvc, err := fakeController.kubeClientset.CoreV1().Services(gateway.Namespace).Get("wenhook-gateway-svc", metav1.GetOptions{})
							convey.So(err, convey.ShouldNotBeNil)
							convey.So(gatewaySvc, convey.ShouldBeNil)
						})
					})
				})
			})
		})
	})
}
