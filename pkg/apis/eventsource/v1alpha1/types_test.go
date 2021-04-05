package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetReplicas(t *testing.T) {
	ep := EventSourceSpec{}
	assert.Equal(t, ep.GetReplicas(), int32(1))
	ep.Replicas = convertInt(t, 0)
	assert.Equal(t, ep.GetReplicas(), int32(1))
	ep.Replicas = convertInt(t, 2)
	assert.Equal(t, ep.GetReplicas(), int32(2))
	ep.Replicas = nil
	ep.DeprecatedReplica = convertInt(t, 0)
	assert.Equal(t, ep.GetReplicas(), int32(1))
	ep.DeprecatedReplica = convertInt(t, 1)
	assert.Equal(t, ep.GetReplicas(), int32(1))
	ep.DeprecatedReplica = convertInt(t, 2)
	assert.Equal(t, ep.GetReplicas(), int32(2))
}

func convertInt(t *testing.T, num int) *int32 {
	t.Helper()
	r := int32(num)
	return &r
}
