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
	"testing"

	"github.com/argoproj/argo-events/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSensorControllerConfigWatch(t *testing.T) {
	sensorController := getController()
	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.DefaultControllerNamespace,
			Name:      sensorController.ConfigMap,
		},
		Data: map[string]string{
			common.ControllerConfigMapKey: `instanceID: fake-instance-id`,
		},
	}
	cm, err := sensorController.k8sClient.CoreV1().ConfigMaps(sensorController.Namespace).Create(configmap)
	assert.Nil(t, err)
	assert.NotNil(t, cm)
	err = sensorController.ResyncConfig(sensorController.Namespace)
	assert.Nil(t, err)
	assert.Equal(t, sensorController.Config.InstanceID, "fake-instance-id")
}
