package dependencies

import (
	"testing"
	"time"

	apicommon "github.com/argoproj/argo-events/pkg/apis/common"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFilterContext(t *testing.T) {
	tests := []struct {
		name            string
		expectedContext *apicommon.EventContext
		actualContext   *apicommon.EventContext
		result          bool
	}{
		{
			name: "different event contexts",
			expectedContext: &apicommon.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
			},
			actualContext: &apicommon.EventContext{
				Type:            "calendar",
				SpecVersion:     "0.3",
				Source:          "calendar-gateway",
				ID:              "1",
				Time:            v1.MicroTime{Time: time.Now()},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			result: false,
		},
		{
			name: "contexts are same",
			expectedContext: &apicommon.EventContext{
				Type:   "webhook",
				Source: "webhook-gateway",
			},
			actualContext: &apicommon.EventContext{
				Type:            "webhook",
				SpecVersion:     "0.3",
				Source:          "webhook-gateway",
				ID:              "1",
				Time:            v1.MicroTime{Time: time.Now()},
				DataContentType: "application/json",
				Subject:         "example-1",
			},
			result: true,
		},
		{
			name:            "actual event context is nil",
			expectedContext: &apicommon.EventContext{},
			actualContext:   nil,
			result:          false,
		},
		{
			name:            "expected event context is nil",
			expectedContext: nil,
			result:          true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := filterContext(test.expectedContext, test.actualContext)
			assert.Equal(t, test.result, result)
		})
	}
}
