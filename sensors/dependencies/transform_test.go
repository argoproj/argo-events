package dependencies

import (
	"reflect"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/types"
)

func TestTransform(t *testing.T) {
	subject := "test-subject"

	tests := []struct {
		JQExpr string
		event  *cloudevents.Event
		result *cloudevents.Event
		err    error
	}{
		{
			JQExpr: ".a += 1 | .b *= 2",
			event: &cloudevents.Event{
				Context: &cloudevents.EventContextV1{
					ID:              "hello",
					Source:          types.URIRef{},
					DataContentType: cloudevents.StringOfApplicationJSON(),
					Subject:         &subject,
				},
				DataEncoded: []byte(`{"a":1,"b":2}`),
			},
			result: &cloudevents.Event{
				Context: &cloudevents.EventContextV1{
					ID:              "hello",
					Type:            "test",
					DataContentType: cloudevents.StringOfApplicationJSON(),
					Subject:         &subject,
				},
				DataEncoded: []byte(`{"a":2,"b":4}`),
			},
			err: nil,
		},
	}

	for _, test := range tests {
		result, err := Transform(test.event, test.JQExpr)
		if !reflect.DeepEqual(test.err, err) {
			t.Fatalf("expected: %v, got: %v", test.err, err)
		}
		if !reflect.DeepEqual(test.result.Data(), result.Data()) {
			t.Fatalf("expected: %v, got: %v", test.result.Data(), result.Data())
		}
	}
}
