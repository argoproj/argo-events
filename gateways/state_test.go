package gateways

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/argoproj/argo-events/pkg/apis/gateway/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_markGatewayNodePhase(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	status := gc.markGatewayNodePhase("1234", v1alpha1.NodePhaseInitialized, "init")
	assert.Nil(t, status)

	gc.gw.Status.Nodes = make(map[string]v1alpha1.NodeStatus)

	gc.gw.Status.Nodes["1234"] = v1alpha1.NodeStatus{
		ID: "1234",
		TimeID: "5678",
		Name: "test",
	}

	status = gc.markGatewayNodePhase("1234", v1alpha1.NodePhaseInitialized, "init")
	assert.NotNil(t, status)
	assert.Equal(t, v1alpha1.NodePhaseInitialized, status.Phase)
}

func Test_getNodeByID(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	status := gc.getNodeByID("test")
	assert.Nil(t, status)
	gc.gw.Status.Nodes = make(map[string]v1alpha1.NodeStatus)

	gc.gw.Status.Nodes["test"] = v1alpha1.NodeStatus{
		ID: "1234",
		TimeID: "5678",
		Name: "test",
	}
	status = gc.getNodeByID("test")
	assert.NotNil(t, status)
}

func Test_initializeNode(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	status := gc.initializeNode("test", "test-node", "1234", "init")
	assert.NotNil(t, status)
	assert.Equal(t, string(v1alpha1.NodePhaseInitialized), string(status.Phase))
	status1 := gc.initializeNode("test", "test-node", "1234", "init")
	assert.NotNil(t, status1)
	assert.Equal(t, status, status1)
}

func Test_persistUpdates(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	gc.gw, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gw.Namespace).Create(gw)
	assert.Nil(t, err)
	gc.gw.Spec.Type = "change-type"
	err = gc.persistUpdates()
	assert.Nil(t, err)
	_, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gw.Namespace).Get(gc.gw.Name, metav1.GetOptions{})
	assert.Equal(t, "change-type", gc.gw.Spec.Type)
}

func Test_reapplyupdate(t *testing.T) {
	gw, err := getGateway()
	assert.Nil(t, err)
	assert.NotNil(t, gw)
	gc := newGatewayconfig(gw)
	gc.gw, err = gc.gwcs.ArgoprojV1alpha1().Gateways(gw.Namespace).Create(gw)
	assert.Nil(t, err)
	gc.gw.Spec.Type = "change-type"
	err = gc.reapplyUpdate()
	assert.Nil(t, err)
}
