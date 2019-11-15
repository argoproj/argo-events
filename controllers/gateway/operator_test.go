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

	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
	"github.com/ghodss/yaml"
	"github.com/smartystreets/goconvey/convey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var testGatewayStr = `apiVersion: argoproj.io/v1alpha1
kind: Gateway
metadata:
 name: webhook-gateway
 namespace: argo-events
 labels:
   gateways.argoproj.io/gateway-controller-instanceid: argo-events
   gateway-name: "webhook-gateway"
   argo-events-gateway-version: v0.10
spec:
  eventSource: "webhook-gateway-configmap"
  type: "webhook"
  processorPort: "9330"
  eventProtocol:
    type: "NATS"
    nats:
      url: "nats://nats.argo-events:4222"
      type: "Standard"
  eventVersion: "1.0"
  template:
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
  service:
    metadata:
      name: webhook-gateway-svc
    spec:
      selector:
        gateway-name: "webhook-gateway"
      ports:
        - port: 12000
          targetPort: 12000
      type: LoadBalancer`

var (
	gatewayPodName = "webhook-gateway"
	gatewaySvcName = "webhook-gateway-svc"
)

func getGateway() (*v1alpha1.Gateway, error) {
	gwBytes := []byte(testGatewayStr)
	var gateway v1alpha1.Gateway
	err := yaml.Unmarshal(gwBytes, &gateway)
	return &gateway, err
}

func waitForAllInformers(done chan struct{}, controller *Controller) {
	cache.WaitForCacheSync(done, controller.informer.HasSynced)
	cache.WaitForCacheSync(done, controller.podInformer.Informer().HasSynced)
	cache.WaitForCacheSync(done, controller.svcInformer.Informer().HasSynced)
}

func getPodAndService(controller *Controller, namespace string) (*corev1.Pod, *corev1.Service, error) {
	pod, err := controller.k8sClient.CoreV1().Pods(namespace).Get(gatewayPodName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	svc, err := controller.k8sClient.CoreV1().Services(namespace).Get(gatewaySvcName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	return pod, svc, err
}

func deletePodAndService(controller *Controller, namespace string) error {
	err := controller.k8sClient.CoreV1().Pods(namespace).Delete(gatewayPodName, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	err = controller.k8sClient.CoreV1().Services(namespace).Delete(gatewaySvcName, &metav1.DeleteOptions{})
	return err
}

func TestGatewayOperateLifecycle(t *testing.T) {
	done := make(chan struct{})
	convey.Convey("Given a gateway resource spec, parse it", t, func() {
		fakeController := getGatewayController()
		gateway, err := getGateway()
		convey.Convey("Make sure no error occurs", func() {
			convey.So(err, convey.ShouldBeNil)

			convey.Convey("Create the gateway", func() {
				gateway, err = fakeController.gatewayClient.ArgoprojV1alpha1().Gateways(fakeController.Config.Namespace).Create(gateway)

				convey.Convey("No error should occur and gateway resource should not be empty", func() {
					convey.So(err, convey.ShouldBeNil)
					convey.So(gateway, convey.ShouldNotBeNil)

					convey.Convey("Create a new gateway operation context", func() {
						goc := newOperationCtx(gateway, fakeController)
						convey.So(goc, convey.ShouldNotBeNil)

						convey.Convey("Operate on new gateway", func() {
							goc.markGatewayPhase(v1alpha1.NodePhaseNew, "test")
							err := goc.operate()

							convey.Convey("Operation must succeed", func() {
								convey.So(err, convey.ShouldBeNil)
								waitForAllInformers(done, fakeController)

								convey.Convey("A gateway pod and service must be created", func() {
									gatewayPod, gatewaySvc, err := getPodAndService(fakeController, gateway.Namespace)
									convey.So(err, convey.ShouldBeNil)
									convey.So(gatewayPod, convey.ShouldNotBeNil)
									convey.So(gatewaySvc, convey.ShouldNotBeNil)

									convey.Convey("Go to running state", func() {
										gateway, err := fakeController.gatewayClient.ArgoprojV1alpha1().Gateways(gateway.Namespace).Get(gateway.Name, metav1.GetOptions{})
										convey.So(err, convey.ShouldBeNil)
										convey.So(gateway.Status.Phase, convey.ShouldEqual, v1alpha1.NodePhaseRunning)
									})
								})
							})
						})

						convey.Convey("Operate on gateway in running state", func() {
							err := fakeController.gatewayClient.ArgoprojV1alpha1().Gateways(gateway.Namespace).Delete(gateway.Name, &metav1.DeleteOptions{})
							convey.So(err, convey.ShouldBeNil)
							gateway, err = fakeController.gatewayClient.ArgoprojV1alpha1().Gateways(gateway.Namespace).Create(gateway)
							convey.So(err, convey.ShouldBeNil)
							convey.So(gateway, convey.ShouldNotBeNil)

							goc.markGatewayPhase(v1alpha1.NodePhaseNew, "test")

							// Operate it once to create pod and service
							waitForAllInformers(done, fakeController)
							err = goc.operate()
							convey.So(err, convey.ShouldBeNil)
							waitForAllInformers(done, fakeController)

							convey.Convey("Operation must succeed", func() {
								goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "test")

								waitForAllInformers(done, fakeController)
								err := goc.operate()
								convey.So(err, convey.ShouldBeNil)
								waitForAllInformers(done, fakeController)

								convey.Convey("Untouch pod and service", func() {
									gatewayPod, gatewaySvc, err := getPodAndService(fakeController, gateway.Namespace)
									convey.So(err, convey.ShouldBeNil)
									convey.So(gatewayPod, convey.ShouldNotBeNil)
									convey.So(gatewaySvc, convey.ShouldNotBeNil)

									convey.Convey("Stay in running state", func() {
										gateway, err := fakeController.gatewayClient.ArgoprojV1alpha1().Gateways(gateway.Namespace).Get(gateway.Name, metav1.GetOptions{})
										convey.So(err, convey.ShouldBeNil)
										convey.So(gateway.Status.Phase, convey.ShouldEqual, v1alpha1.NodePhaseRunning)
									})
								})
							})

							convey.Convey("Delete pod and service", func() {
								err := deletePodAndService(fakeController, gateway.Namespace)
								convey.So(err, convey.ShouldBeNil)

								convey.Convey("Operation must succeed", func() {
									goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "test")

									waitForAllInformers(done, fakeController)
									err := goc.operate()
									convey.So(err, convey.ShouldBeNil)
									waitForAllInformers(done, fakeController)

									convey.Convey("Create pod and service", func() {
										gatewayPod, gatewaySvc, err := getPodAndService(fakeController, gateway.Namespace)
										convey.So(err, convey.ShouldBeNil)
										convey.So(gatewayPod, convey.ShouldNotBeNil)
										convey.So(gatewaySvc, convey.ShouldNotBeNil)

										convey.Convey("Stay in running state", func() {
											gateway, err := fakeController.gatewayClient.ArgoprojV1alpha1().Gateways(gateway.Namespace).Get(gateway.Name, metav1.GetOptions{})
											convey.So(err, convey.ShouldBeNil)
											convey.So(gateway.Status.Phase, convey.ShouldEqual, v1alpha1.NodePhaseRunning)
										})
									})
								})
							})

							convey.Convey("Change pod and service spec", func() {
								goc.gatewayObj.Spec.Template.Spec.RestartPolicy = "Never"
								goc.gatewayObj.Spec.Service.Spec.ClusterIP = "127.0.0.1"

								gatewayPod, gatewaySvc, err := getPodAndService(fakeController, gateway.Namespace)
								convey.So(err, convey.ShouldBeNil)
								convey.So(gatewayPod, convey.ShouldNotBeNil)
								convey.So(gatewaySvc, convey.ShouldNotBeNil)

								convey.Convey("Operation must succeed", func() {
									goc.markGatewayPhase(v1alpha1.NodePhaseRunning, "test")

									waitForAllInformers(done, fakeController)
									err := goc.operate()
									convey.So(err, convey.ShouldBeNil)
									waitForAllInformers(done, fakeController)

									convey.Convey("Delete pod and service", func() {
										gatewayPod, gatewaySvc, err := getPodAndService(fakeController, gateway.Namespace)
										convey.So(err, convey.ShouldBeNil)
										convey.So(gatewayPod.Spec.RestartPolicy, convey.ShouldEqual, "Never")
										convey.So(gatewaySvc.Spec.ClusterIP, convey.ShouldEqual, "127.0.0.1")

										convey.Convey("Stay in running state", func() {
											gateway, err := fakeController.gatewayClient.ArgoprojV1alpha1().Gateways(gateway.Namespace).Get(gateway.Name, metav1.GetOptions{})
											convey.So(err, convey.ShouldBeNil)
											convey.So(gateway.Status.Phase, convey.ShouldEqual, v1alpha1.NodePhaseRunning)
										})
									})
								})
							})
						})

						convey.Convey("Operate on gateway in error state", func() {
							err := fakeController.gatewayClient.ArgoprojV1alpha1().Gateways(gateway.Namespace).Delete(gateway.Name, &metav1.DeleteOptions{})
							convey.So(err, convey.ShouldBeNil)
							gateway, err = fakeController.gatewayClient.ArgoprojV1alpha1().Gateways(gateway.Namespace).Create(gateway)
							convey.So(err, convey.ShouldBeNil)
							convey.So(gateway, convey.ShouldNotBeNil)

							goc.markGatewayPhase(v1alpha1.NodePhaseNew, "test")

							// Operate it once to create pod and service
							waitForAllInformers(done, fakeController)
							err = goc.operate()
							convey.So(err, convey.ShouldBeNil)
							waitForAllInformers(done, fakeController)

							convey.Convey("Operation must succeed", func() {
								goc.markGatewayPhase(v1alpha1.NodePhaseError, "test")

								waitForAllInformers(done, fakeController)
								err := goc.operate()
								convey.So(err, convey.ShouldBeNil)
								waitForAllInformers(done, fakeController)

								convey.Convey("Untouch pod and service", func() {
									gatewayPod, gatewaySvc, err := getPodAndService(fakeController, gateway.Namespace)
									convey.So(err, convey.ShouldBeNil)
									convey.So(gatewayPod, convey.ShouldNotBeNil)
									convey.So(gatewaySvc, convey.ShouldNotBeNil)

									convey.Convey("Stay in error state", func() {
										gateway, err := fakeController.gatewayClient.ArgoprojV1alpha1().Gateways(gateway.Namespace).Get(gateway.Name, metav1.GetOptions{})
										convey.So(err, convey.ShouldBeNil)
										convey.So(gateway.Status.Phase, convey.ShouldEqual, v1alpha1.NodePhaseError)
									})
								})
							})

							convey.Convey("Delete pod and service", func() {
								err := deletePodAndService(fakeController, gateway.Namespace)
								convey.So(err, convey.ShouldBeNil)

								convey.Convey("Operation must succeed", func() {
									goc.markGatewayPhase(v1alpha1.NodePhaseError, "test")

									waitForAllInformers(done, fakeController)
									err := goc.operate()
									convey.So(err, convey.ShouldBeNil)
									waitForAllInformers(done, fakeController)

									convey.Convey("Create pod and service", func() {
										gatewayPod, gatewaySvc, err := getPodAndService(fakeController, gateway.Namespace)
										convey.So(err, convey.ShouldBeNil)
										convey.So(gatewayPod, convey.ShouldNotBeNil)
										convey.So(gatewaySvc, convey.ShouldNotBeNil)

										convey.Convey("Go to running state", func() {
											gateway, err := fakeController.gatewayClient.ArgoprojV1alpha1().Gateways(gateway.Namespace).Get(gateway.Name, metav1.GetOptions{})
											convey.So(err, convey.ShouldBeNil)
											convey.So(gateway.Status.Phase, convey.ShouldEqual, v1alpha1.NodePhaseRunning)
										})
									})
								})
							})

							convey.Convey("Change pod and service spec", func() {
								goc.gatewayObj.Spec.Template.Spec.RestartPolicy = "Never"
								goc.gatewayObj.Spec.Service.Spec.ClusterIP = "127.0.0.1"

								gatewayPod, gatewaySvc, err := getPodAndService(fakeController, gateway.Namespace)
								convey.So(err, convey.ShouldBeNil)
								convey.So(gatewayPod, convey.ShouldNotBeNil)
								convey.So(gatewaySvc, convey.ShouldNotBeNil)

								convey.Convey("Operation must succeed", func() {
									goc.markGatewayPhase(v1alpha1.NodePhaseError, "test")

									waitForAllInformers(done, fakeController)
									err := goc.operate()
									convey.So(err, convey.ShouldBeNil)
									waitForAllInformers(done, fakeController)

									convey.Convey("Delete pod and service", func() {
										gatewayPod, gatewaySvc, err := getPodAndService(fakeController, gateway.Namespace)
										convey.So(err, convey.ShouldBeNil)
										convey.So(gatewayPod.Spec.RestartPolicy, convey.ShouldEqual, "Never")
										convey.So(gatewaySvc.Spec.ClusterIP, convey.ShouldEqual, "127.0.0.1")

										convey.Convey("Go to running state", func() {
											gateway, err := fakeController.gatewayClient.ArgoprojV1alpha1().Gateways(gateway.Namespace).Get(gateway.Name, metav1.GetOptions{})
											convey.So(err, convey.ShouldBeNil)
											convey.So(gateway.Status.Phase, convey.ShouldEqual, v1alpha1.NodePhaseRunning)
										})
									})
								})
							})
						})
					})
				})
			})
		})
	})
}

func TestPersistUpdates(t *testing.T) {
	convey.Convey("Given a gateway resource", t, func() {
		namespace := "argo-events"
		client := fake.NewSimpleClientset()
		logger := common.NewArgoEventsLogger()
		gw, err := getGateway()
		convey.So(err, convey.ShouldBeNil)

		convey.Convey("Create the gateway", func() {
			gw, err = client.ArgoprojV1alpha1().Gateways(namespace).Create(gw)
			convey.So(err, convey.ShouldBeNil)
			convey.So(gw, convey.ShouldNotBeNil)

			gw.ObjectMeta.Labels = map[string]string{
				"default": "default",
			}

			convey.Convey("Update the gateway", func() {
				updatedGw, err := PersistUpdates(client, gw, logger)
				convey.So(err, convey.ShouldBeNil)
				convey.So(updatedGw, convey.ShouldNotEqual, gw)
				convey.So(updatedGw.Labels, convey.ShouldResemble, gw.Labels)

				updatedGw.Labels["new"] = "new"

				convey.Convey("Reapply the gateway", func() {
					err := ReapplyUpdates(client, updatedGw)
					convey.So(err, convey.ShouldBeNil)
					convey.So(len(updatedGw.Labels), convey.ShouldEqual, 2)
				})
			})
		})
	})
}
