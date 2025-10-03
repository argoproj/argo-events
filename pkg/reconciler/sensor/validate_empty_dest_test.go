package sensor

import (
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/argoproj/argo-events/pkg/sensors/triggers"
	"github.com/stretchr/testify/assert"
)

func TestHTTPTriggerWithEmptyDest(t *testing.T) {
	// Test HTTP trigger validation with empty dest
	httpTrigger := &v1alpha1.HTTPTrigger{
		URL:    "http://example.com",
		Method: "POST",
		Payload: []v1alpha1.TriggerParameter{
			{
				Src: &v1alpha1.TriggerParameterSource{
					DependencyName: "test-dep",
					UseRawData:     true,
				},
				Dest: "", // Empty dest - this should now be allowed for HTTP triggers
			},
		},
	}

	// Test validation - should pass
	err := validateHTTPTrigger(httpTrigger)
	assert.NoError(t, err, "HTTP trigger validation should pass with empty dest")

	// Test ConstructPayload with empty dest
	events := map[string]*v1alpha1.Event{
		"test-dep": {
			Context: &v1alpha1.EventContext{
				Type:            "test",
				Source:          "test",
				DataContentType: "application/json",
			},
			Data: []byte(`{"message": "hello world"}`),
		},
	}

	payload, err := triggers.ConstructPayload(events, httpTrigger.Payload)
	assert.NoError(t, err, "ConstructPayload should succeed with empty dest")

	// For empty dest with UseRawData, we expect the entire event object as JSON
	assert.NotEmpty(t, payload, "Payload should not be empty")
	assert.Contains(t, string(payload), "context", "Payload should contain event context")
	assert.Contains(t, string(payload), "data", "Payload should contain event data")
}

func TestNonHTTPTriggerWithEmptyDestShouldFail(t *testing.T) {
	// Test that non-HTTP triggers still require dest
	parameter := &v1alpha1.TriggerParameter{
		Src: &v1alpha1.TriggerParameterSource{
			DependencyName: "test-dep",
			DataKey:        "body",
		},
		Dest: "", // Empty dest - this should fail for non-HTTP triggers
	}

	err := validateTriggerParameter(parameter)
	assert.Error(t, err, "Non-HTTP trigger parameter validation should fail with empty dest")
	assert.Contains(t, err.Error(), "parameter destination can't be empty")
}
