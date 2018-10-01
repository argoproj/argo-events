package gateway

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGwOperationCtx_validate(t *testing.T) {
	fakeController := getGatewayController()
	gateway, err := getGateway()
	assert.Nil(t, err)
	goc := newGatewayOperationCtx(gateway, fakeController)
	err = Validate(goc.gw)
	assert.Nil(t, err)
}
