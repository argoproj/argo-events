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
	"context"
	"github.com/argoproj/argo-events/common"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var (
	configmapName = common.DefaultConfigMapName("gateway-controller")
)

func TestWatchControllerConfigMap(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gc := getGatewayController()
	_, err := gc.watchControllerConfigMap(ctx)
	assert.Nil(t, err)
}

func TestNewControllerConfigMapWatch(t *testing.T) {
	gc := getGatewayController()
	watcher := gc.newControllerConfigMapWatch()
	assert.NotNil(t, watcher)
}

func TestGatewayController_ResyncConfig(t *testing.T) {
	gc := getGatewayController()
	cmObj := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.DefaultControllerNamespace,
			Name:      gc.ConfigMap,
		},
		Data: map[string]string{
			common.GatewayControllerConfigMapKey: `instanceID: argo-events`,
		},
	}

	cm, err := gc.kubeClientset.CoreV1().ConfigMaps(gc.Namespace).Create(cmObj)
	assert.Nil(t, err)

	err = gc.ResyncConfig(gc.Namespace)
	assert.Nil(t, err)
	assert.NotNil(t, cm)
	assert.NotNil(t, gc.Config)
	assert.NotEqual(t, gc.Config.Namespace, gc.Namespace)
	assert.Equal(t, gc.Config.Namespace, corev1.NamespaceAll)
}
