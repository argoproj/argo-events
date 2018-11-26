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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var sampleGatewayStr = `apiVersion: argoproj.io/v1alpha1
kind: Gateway
metadata:
  name: webhook-gateway
  labels:
    gateways.argoproj.io/gateway-controller-instanceid: argo-events
    gateway-name: "webhook-gateway"
spec:
  configMap: "webhook-gateway-configmap"
  type: "webhook"
  dispatchMechanism: "HTTP"
  eventVersion: "1.0"
  imageVersion: "latest"
  deploySpec:
    containers:
    - name: "webhook-events"
      image: "argoproj/webhook-gateway"
      imagePullPolicy: "Always"
      command: ["/bin/webhook-gateway"]
    serviceAccountName: "argo-events-sa"
  serviceSpec:
    selector:
      gateway-name: "webhook-gateway"
    ports:
      - port: 12000
        targetPort: 12000
    type: LoadBalancer
  watchers:
    gateways:
    - name: "webhook-gateway-2"
      port: "9070"
      endpoint: "/notifications"
    sensors:
    - name: "webhook-sensor"
    - name: "multi-signal-sensor"
    - name: "webhook-time-filter-sensor"`

func getGateway() (*v1alpha1.Gateway, error) {
	gwBytes := []byte(sampleGatewayStr)
	var gateway v1alpha1.Gateway
	err := yaml.Unmarshal(gwBytes, &gateway)
	return &gateway, err
}

func TestGatewayOperateLifecycle(t *testing.T) {
	fakeController := getGatewayController()

	gateway, err := getGateway()
	assert.Nil(t, err)

	gateway, err = fakeController.gatewayClientset.ArgoprojV1alpha1().Gateways(fakeController.Config.Namespace).Create(gateway)
	assert.Nil(t, err)

	goc := newGatewayOperationCtx(gateway, fakeController)

	// STEP 1: operate on new gateway
	err = goc.operate()
	assert.Nil(t, err)

	// assert the status of sensor's signal is initializing
	assert.Equal(t, string(v1alpha1.NodePhaseRunning), string(goc.gw.Status.Phase))
	assert.Equal(t, 0, len(goc.gw.Status.Nodes))

	// check whether gateway transformer configmap is created
	cm, err := fakeController.kubeClientset.CoreV1().ConfigMaps(fakeController.Config.Namespace).Get(common.DefaultGatewayTransformerConfigMapName(goc.gw.ObjectMeta.Name), metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, cm.Data[common.EventTypeVersion], goc.gw.Spec.EventVersion)
	assert.Equal(t, cm.Data[common.EventSource], goc.gw.ObjectMeta.Name)
	assert.Equal(t, cm.Data[common.EventType], string(goc.gw.Spec.Type))
	assert.NotNil(t, cm.Data[common.SensorWatchers])
	assert.NotNil(t, cm.Data[common.GatewayWatchers])

	// check whether gateway deployment is successful
	gwDeployment, err := fakeController.kubeClientset.AppsV1().Deployments(fakeController.Config.Namespace).Get(common.DefaultGatewayDeploymentName(goc.gw.ObjectMeta.Name), metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, gwDeployment)

	// check whether gateway service is created
	gwService, err := fakeController.kubeClientset.CoreV1().Services(fakeController.Config.Namespace).Get(common.DefaultGatewayServiceName(goc.gw.ObjectMeta.Name), metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, gwService)
	assert.Equal(t, string(gwService.Spec.Type), string(corev1.ServiceTypeLoadBalancer))

	// mark gateway phase as error
	goc.gw.Status.Phase = v1alpha1.NodePhaseError
	err = goc.operate()
	assert.Nil(t, err)
	assert.Equal(t, string(goc.gw.Status.Phase), string(v1alpha1.NodePhaseError))

	// mark gateway phase as service error
	goc.gw.Status.Phase = v1alpha1.NodePhaseServiceError
	err = goc.operate()
	assert.Nil(t, err)
	assert.Equal(t, string(goc.gw.Status.Phase), string(v1alpha1.NodePhaseRunning))

	// operate on gateway that is already running
	err = goc.operate()
	assert.Nil(t, err)
	assert.Equal(t, string(goc.gw.Status.Phase), string(v1alpha1.NodePhaseRunning))
}
