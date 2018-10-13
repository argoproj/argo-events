package gateway

import (
	"github.com/argoproj/argo-events/common"
	fake_gw "github.com/argoproj/argo-events/pkg/client/gateway/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/selection"
	"testing"
)

func TestInstanceIDReq(t *testing.T) {
	controller := &GatewayController{
		Config: GatewayControllerConfig{
			InstanceID: "argo-events",
		},
	}

	req := controller.instanceIDReq()
	assert.Equal(t, common.LabelKeyGatewayControllerInstanceID, req.Key())

	req = controller.instanceIDReq()
	assert.Equal(t, selection.Equals, req.Operator())
	assert.True(t, req.Values().Has("argo-events"))
}

func TestNewGatewayInformer(t *testing.T) {
	controller := &GatewayController{
		Config: GatewayControllerConfig{
			Namespace:  "testing",
			InstanceID: "argo-events",
		},
		gatewayClientset: fake_gw.NewSimpleClientset(),
	}
	assert.NotNil(t, controller.newGatewayInformer())
}
