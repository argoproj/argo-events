package dependencies

import (
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func strptr(s string) *string {
	return &s
}

func TestApplyJQTransform(t *testing.T) {
	tests := []struct {
		event    *cloudevents.Event
		result   *cloudevents.Event
		command  string
		hasError bool
	}{
		{
			event: &cloudevents.Event{
				Context: &cloudevents.EventContextV1{
					ID:              "123",
					Source:          types.URIRef{},
					DataContentType: strptr(cloudevents.ApplicationJSON),
					Subject:         strptr("hello"),
					Time:            &types.Timestamp{},
				},
				DataEncoded: []byte(`{"a":1,"b":"2"}`),
			},
			result: &cloudevents.Event{
				Context: &cloudevents.EventContextV1{
					ID:              "123",
					Source:          types.URIRef{},
					DataContentType: strptr(cloudevents.ApplicationJSON),
					Subject:         strptr("hello"),
					Time:            &types.Timestamp{},
				},
				DataEncoded: []byte(`{"a":2,"b":"22"}`),
			},
			hasError: false,
			command:  ".a += 1 | .b *= 2",
		},
	}
	for _, tt := range tests {
		result, err := applyJQTransform(tt.event, tt.command)
		if tt.hasError {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}
		assert.Equal(t, tt.result.Data(), result.Data())
	}
}

func TestApplyScriptTransform(t *testing.T) {

}
