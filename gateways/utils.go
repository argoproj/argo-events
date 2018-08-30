package gateways

import (
	"github.com/argoproj/argo-events/controllers/gateway/transform"
	"encoding/json"
)

func CreateTransformPayload(b []byte, source string) ([]byte, error) {
	tp := &transform.TransformerPayload{
		Src: source,
		Payload: b,
	}
	payload, err := json.Marshal(tp)
	if err != nil {
		return nil, err
	}
	return payload, nil
}
