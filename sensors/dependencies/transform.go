package dependencies

import (
	"encoding/json"
	"fmt"

	"github.com/argoproj/argo-events/pkg/apis/sensor/v1alpha1"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/itchyny/gojq"
	"github.com/tidwall/gjson"
	lua "github.com/yuin/gopher-lua"
)

func ApplyTransform(event *cloudevents.Event, transform *v1alpha1.EventDependencyTransformer) (*cloudevents.Event, error) {
	if transform == nil {
		return event, nil
	}
	if transform.JQ != "" {
		return applyJQTransform(event, transform.JQ)
	}
	if transform.Script != "" {
		return applyScriptTransform(event, transform.Script)
	}
	return event, nil
}

func applyJQTransform(event *cloudevents.Event, command string) (*cloudevents.Event, error) {
	if event == nil {
		return nil, fmt.Errorf("nil Event")
	}
	payload := event.Data()
	if payload == nil {
		return event, nil
	}
	var js *json.RawMessage
	if err := json.Unmarshal(payload, &js); err != nil {
		return nil, err
	}
	var jsData []byte
	jsData, err := json.Marshal(js)
	if err != nil {
		return nil, err
	}
	query, err := gojq.Parse(command)
	if err != nil {
		return nil, err
	}
	var temp map[string]interface{}
	if err = json.Unmarshal(jsData, &temp); err != nil {
		return nil, err
	}
	iter := query.Run(temp)
	v, ok := iter.Next()
	if !ok {
		return nil, fmt.Errorf("no output available from the jq command execution")
	}
	switch v.(type) {
	case map[string]interface{}:
		resultContent, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		if !gjson.ValidBytes(resultContent) {
			return nil, fmt.Errorf("jq transformation output is not a JSON object")
		}
		if err = event.SetData(cloudevents.ApplicationJSON, resultContent); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("jq transformation output must be a JSON object")
	}
	return event, nil
}

func applyScriptTransform(event *cloudevents.Event, script string) (*cloudevents.Event, error) {
	l := lua.NewState()
	defer l.Close()
	payload := event.Data()
	if payload == nil {
		return event, nil
	}
	var js *json.RawMessage
	if err := json.Unmarshal(payload, &js); err != nil {
		return nil, err
	}
	var jsData []byte
	jsData, err := json.Marshal(js)
	if err != nil {
		return nil, err
	}
	l.SetGlobal("event", lua.LString(string(jsData)))
	if err = l.DoString(script); err != nil {
		return nil, err
	}
	lv := l.Get(-1)
	result, ok := lv.(lua.LString)
	if !ok {
		return nil, fmt.Errorf("transformation result type is not of string")
	}
	if !gjson.Valid(result.String()) {
		return nil, fmt.Errorf("script transformation output is not a JSON object")
	}
	if err = event.SetData(cloudevents.ApplicationJSON, []byte(result.String())); err != nil {
		return nil, err
	}
	return event, nil
}
