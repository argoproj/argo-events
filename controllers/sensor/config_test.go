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

func TestResyncConfig(t *testing.T) {
	controller := sensorController()
	err := controller.ResyncConfig(common.DefaultControllerNamespace)
	assert.NotNil(t, err)

	// fail when the configmap does not have key 'config'
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SensorControllerConfigmap,
			Namespace: common.DefaultControllerNamespace,
		},
		Data: map[string]string{},
	}
	_, err = controller.kubeClientset.CoreV1().ConfigMaps(common.DefaultControllerNamespace).Create(configMap)
	assert.Nil(t, err)
	err = controller.ResyncConfig(common.DefaultControllerNamespace)
	assert.NotNil(t, err)

	// succeed with no errors now that configmap has 'config' key
	configMap.Data = map[string]string{"config": controllerConfig}
	_, err = controller.kubeClientset.CoreV1().ConfigMaps(common.DefaultControllerNamespace).Update(configMap)
	assert.Nil(t, err)
	err = controller.ResyncConfig(common.DefaultControllerNamespace)
	assert.Nil(t, err)
}

var controllerConfig = `
instanceID: argo-events
`
