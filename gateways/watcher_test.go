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
	e := gc.GetK8Event("test", v1alpha1.NodePhaseCompleted, &ConfigData{
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
