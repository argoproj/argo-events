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
	"github.com/argoproj/argo-events/common"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_filterEvents(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	e := gc.GetK8Event("test", v1alpha1.NodePhaseCompleted, &EventSourceData{
		Config: "testConfig",
		Src:    "testSrc",
		ID:     "1234",
		TimeID: "4567",
	})
	ok := gc.filterEvent(e)
	assert.Equal(t, true, ok)

	e.Labels[common.LabelEventSeen] = "true"
	ok = gc.filterEvent(e)
	assert.Equal(t, false, ok)

	e.Labels[common.LabelEventSeen] = ""
	e.Source.Component = "test"
	ok = gc.filterEvent(e)
	assert.Equal(t, false, ok)

	e.Source.Component = gc.gw.Name
	ok = gc.filterEvent(e)
	assert.Equal(t, true, ok)

	e.ReportingInstance = "testI"
	ok = gc.filterEvent(e)
	assert.Equal(t, false, ok)
	e.ReportingInstance = gc.controllerInstanceID
	e.ReportingController = "testC"
	ok = gc.filterEvent(e)
	assert.Equal(t, false, ok)
}
