package eventsources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNeedsLeaderElection(t *testing.T) {
	tests := []struct {
		name           string
		isRecreateType bool
		replicas       int
		want           bool
	}{
		{"non-recreate type, single replica", false, 1, false},
		{"non-recreate type, multiple replicas", false, 3, false},
		{"recreate type, single replica", true, 1, false},
		{"recreate type, zero replicas", true, 0, false},
		{"recreate type, multiple replicas", true, 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, needsLeaderElection(tt.isRecreateType, tt.replicas))
		})
	}
}
