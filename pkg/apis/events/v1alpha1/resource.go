package v1alpha1

import "encoding/json"

/**
This inspired by intstr.IntOrStr and json.RawMessage.
*/

// K8SResource represent arbitrary structured data.
type K8SResource struct {
	Value []byte `json:"value" protobuf:"bytes,1,opt,name=value"`
}

func NewK8SResource(s interface{}) K8SResource {
	data, _ := json.Marshal(s)
	return K8SResource{Value: data}
}

func (a *K8SResource) UnmarshalJSON(value []byte) error {
	a.Value = value
	return nil
}

func (n K8SResource) MarshalJSON() ([]byte, error) {
	return n.Value, nil
}

func (n K8SResource) OpenAPISchemaType() []string {
	return []string{"object"}
}

func (n K8SResource) OpenAPISchemaFormat() string { return "" }
