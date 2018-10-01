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

package sensor

import (
	"context"
	"os"
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestWatchControllerConfigMap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	controller := SensorController{
		ConfigMap:     "sensor-controller-configmap",
		ConfigMapNS:   "testing",
		kubeClientset: fake.NewSimpleClientset(),
	}
	_, err := controller.watchControllerConfigMap(ctx)
	assert.Nil(t, err)
}

func TestNewControllerConfigMapWatch(t *testing.T) {
	controller := SensorController{
		ConfigMap:     "sensor-controller-configmap",
		ConfigMapNS:   "testing",
		kubeClientset: fake.NewSimpleClientset(),
	}
	controller.newControllerConfigMapWatch()
}

func TestResyncConfig(t *testing.T) {
	defer os.Unsetenv(common.EnvVarNamespace)
	controller := SensorController{
		ConfigMap:     "sensor-controller-configmap",
		ConfigMapNS:   "testing",
		kubeClientset: fake.NewSimpleClientset(),
	}

	os.Setenv(common.EnvVarNamespace, "testing")

	err := controller.ResyncConfig("testing")
	assert.NotNil(t, err)

	// Note: need to refresh the namespace
	common.RefreshNamespace()

	// fail when the configmap does not have key 'config'
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sensor-controller-configmap",
			Namespace: "testing",
		},
		Data: map[string]string{},
	}
	_, err = controller.kubeClientset.CoreV1().ConfigMaps("testing").Create(configMap)
	assert.Nil(t, err)
	err = controller.ResyncConfig("testing")
	assert.NotNil(t, err)

	// succeed with no errors now that configmap has 'config' key
	configMap.Data = map[string]string{"config": controllerConfig}
	_, err = controller.kubeClientset.CoreV1().ConfigMaps("testing").Update(configMap)
	assert.Nil(t, err)
	err = controller.ResyncConfig("testing")
	assert.Nil(t, err)
}

var controllerConfig = `
instanceID: argo-events
executorImage: argoproj/sensor-controller:latest
`
