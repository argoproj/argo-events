package dependencies

import (
	"testing"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestScriptFilter(t *testing.T) {
	tests := []struct {
		script   string
		event    *v1alpha1.Event
		result   bool
		hasError bool
	}{
		{
			script: `
if event.a == "hello" then return true else return false end
`,
			event: &v1alpha1.Event{
				Data: []byte(`{"a":"hello"}`),
			},
			result:   true,
			hasError: false,
		},
		{
			script: `
if event.a == "hello" and event.b == "world" then return false else return true end
`,
			event: &v1alpha1.Event{
				Data: []byte(`{"a":"hello","b":"world"}`),
			},
			result:   false,
			hasError: false,
		},
		{
			script: `
if event.a == "hello" return false else return true end
`,
			event: &v1alpha1.Event{
				Data: []byte(`{"a":"hello"}`),
			},
			result:   false,
			hasError: true,
		},
		{
			script: `
if a.a == "hello" then return true else return false end
`,
			event: &v1alpha1.Event{
				Data: []byte(`{"a":"hello"}`),
			},
			result:   false,
			hasError: true,
		},
	}
	for _, tt := range tests {
		result, err := filterScript(tt.script, tt.event)
		if tt.hasError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, tt.result, result)
		}
	}
}
