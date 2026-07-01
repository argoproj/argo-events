package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvent_DataString(t *testing.T) {
	assert.Empty(t, Event{}.DataString())
	assert.Equal(t, "bXktZGF0YQ==", Event{Data: []byte("my-data")}.DataString())
	assert.Equal(t, `"my-data"`, Event{Data: []byte(`"my-data"`), Context: &EventContext{DataContentType: "application/json"}}.DataString())
}

func TestGetSensorReplicas(t *testing.T) {
	sp := SensorSpec{}
	assert.Equal(t, sp.GetReplicas(), int32(1))
	sp.Replicas = convertInt(t, 0)
	assert.Equal(t, sp.GetReplicas(), int32(1))
	sp.Replicas = convertInt(t, 2)
	assert.Equal(t, sp.GetReplicas(), int32(2))
}

func TestGRPCTriggerDeepCopy(t *testing.T) {
	original := &GRPCTrigger{
		URL:    "localhost:9000",
		Method: "/helloworld.Greeter/SayHello",
		Schema: []GRPCSchemaField{
			{Name: "message", Number: 1, Type: "string"},
		},
		Payload: []TriggerParameter{
			{Dest: "message"},
		},
		Insecure: true,
		Timeout:  10,
	}

	copied := original.DeepCopy()
	assert.Equal(t, original, copied)

	copied.Schema[0].Name = "changed"
	assert.Equal(t, "message", original.Schema[0].Name, "DeepCopy must not alias the Schema slice")
}

func TestTriggerTemplateGRPCField(t *testing.T) {
	template := &TriggerTemplate{
		Name: "test",
		GRPC: &GRPCTrigger{URL: "localhost:9000", Method: "/pkg.Svc/Method"},
	}
	assert.Equal(t, "localhost:9000", template.GRPC.URL)
}
