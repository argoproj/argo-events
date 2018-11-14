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

package gateways

import (
	"fmt"
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestGatewayConfig_GetK8Event(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	e := gc.GetK8Event("test", "test", &ConfigData{
		Config: "testConfig",
		Src:    "testSrc",
		ID:     "1234",
		TimeID: "4567",
	})
	assert.NotNil(t, e)
	assert.Equal(t, "1234", e.Labels[common.LabelGatewayConfigID])
}

func GetConfigContext() *ConfigContext {
	return &ConfigContext{
		StartChan: make(chan struct{}),
		DataChan:  make(chan []byte),
		ErrChan:   make(chan error),
		DoneChan:  make(chan struct{}),
		StopChan:  make(chan struct{}),
		Data: &ConfigData{
			Config: "testConfig",
			Src:    "testSrc",
			ID:     "1234",
			TimeID: "4567",
		},
	}
}

func TestGatewayConfig_GatewayCleanup(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	ctx := GetConfigContext()
	gc.GatewayCleanup(ctx, nil)
	// check whether k8 event for completion got created
	el, err := gc.Clientset.CoreV1().Events(gw.Namespace).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, el.Items)
	assert.Equal(t, "1234", el.Items[0].Labels[common.LabelGatewayConfigID])
	assert.Equal(t, string(v1alpha1.NodePhaseCompleted), el.Items[0].Action)

	_, ok := <-ctx.DataChan
	assert.Equal(t, false, ok)
	_, ok = <-ctx.StopChan
	assert.Equal(t, false, ok)
	_, ok = <-ctx.DoneChan
	assert.Equal(t, false, ok)
	_, ok = <-ctx.StartChan
	assert.Equal(t, false, ok)
	_, ok = <-ctx.ErrChan
	assert.Equal(t, false, ok)
}

func TestGatewayConfig_GatewayCleanup2(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	ctx := GetConfigContext()
	gc.GatewayCleanup(ctx, fmt.Errorf("error"))
	// check whether k8 event for error got created
	el, err := gc.Clientset.CoreV1().Events(gw.Namespace).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, el.Items)
	assert.Equal(t, "1234", el.Items[0].Labels[common.LabelGatewayConfigID])
	assert.Equal(t, string(v1alpha1.NodePhaseError), el.Items[0].Action)

	_, ok := <-ctx.DataChan
	assert.Equal(t, false, ok)
	_, ok = <-ctx.StopChan
	assert.Equal(t, false, ok)
	_, ok = <-ctx.DoneChan
	assert.Equal(t, false, ok)
	_, ok = <-ctx.StartChan
	assert.Equal(t, false, ok)
	_, ok = <-ctx.ErrChan
	assert.Equal(t, false, ok)
}
