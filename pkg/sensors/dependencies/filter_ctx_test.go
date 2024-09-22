package dependencies

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	sharedutil "github.com/argoproj/argo-events/pkg/shared/util"
)

func TestFilterContext(t *testing.T) {
	tests := []struct {
		name            string
		expectedContext *v1alpha1.EventContext
		actualContext   *v1alpha1.EventContext
		result          bool
	}{
		{
			name: "different event contexts",
			expectedContext: &v1alpha1.EventContext{
				Type: "webhook",
			},
			actualContext: &v1alpha1.EventContext{
				Type:   "calendar",
				Source: "calendar-gateway",
				ID:     "1",
				Time: metav1.Time{
					Time: time.Now().UTC(),
				},
				DataContentType: sharedutil.MediaTypeJSON,
				Subject:         "example-1",
			},
			result: false,
		},
		{
			name: "contexts are same",
			expectedContext: &v1alpha1.EventContext{
				Type:   "webhook",
				Source: "webhook-gateway",
			},
			actualContext: &v1alpha1.EventContext{
				Type:        "webhook",
				SpecVersion: "0.3",
				Source:      "webhook-gateway",
				ID:          "1",
				Time: metav1.Time{
					Time: time.Now().UTC(),
				},
				DataContentType: sharedutil.MediaTypeJSON,
				Subject:         "example-1",
			},
			result: true,
		},
		{
			name:            "actual event context is nil",
			expectedContext: &v1alpha1.EventContext{},
			actualContext:   nil,
			result:          false,
		},
		{
			name:            "expectedResult event context is nil",
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
