package dependencies

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/argoproj/argo-events/pkg/apis/events/v1alpha1"
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
		return event, nil
	default:
		return nil, fmt.Errorf("jq transformation output must be a JSON object")
	}
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
	var payloadJson map[string]interface{}
	if err = json.Unmarshal(jsData, &payloadJson); err != nil {
		return nil, err
	}
	lEvent := mapToTable(payloadJson)
	l.SetGlobal("event", lEvent)
	if err = l.DoString(script); err != nil {
		return nil, err
	}
	lv := l.Get(-1)
	tbl, ok := lv.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("transformation script output type is not of lua table")
	}
	result := toGoValue(tbl)
	resultJson, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	if !gjson.Valid(string(resultJson)) {
		return nil, fmt.Errorf("script transformation output is not a JSON object")
	}
	if err := event.SetData(cloudevents.ApplicationJSON, resultJson); err != nil {
		return nil, err
	}
	return event, nil
}

// MapToTable converts a Go map to a lua table
func mapToTable(m map[string]interface{}) *lua.LTable {
	resultTable := &lua.LTable{}
	for key, element := range m {
		switch t := element.(type) {
		case float64:
			resultTable.RawSetString(key, lua.LNumber(t))
		case int64:
			resultTable.RawSetString(key, lua.LNumber(t))
		case string:
			resultTable.RawSetString(key, lua.LString(t))
		case bool:
			resultTable.RawSetString(key, lua.LBool(t))
		case []byte:
			resultTable.RawSetString(key, lua.LString(string(t)))
		case map[string]interface{}:
			table := mapToTable(element.(map[string]interface{}))
			resultTable.RawSetString(key, table)
		case time.Time:
			resultTable.RawSetString(key, lua.LNumber(t.Unix()))
		case []map[string]interface{}:
			sliceTable := &lua.LTable{}
			for _, s := range element.([]map[string]interface{}) {
				table := mapToTable(s)
				sliceTable.Append(table)
			}
			resultTable.RawSetString(key, sliceTable)
		case []interface{}:
			sliceTable := &lua.LTable{}
			for _, s := range element.([]interface{}) {
				switch tt := s.(type) {
				case map[string]interface{}:
					t := mapToTable(s.(map[string]interface{}))
					sliceTable.Append(t)
				case float64:
					sliceTable.Append(lua.LNumber(tt))
				case string:
					sliceTable.Append(lua.LString(tt))
				case bool:
					sliceTable.Append(lua.LBool(tt))
				}
			}
			resultTable.RawSetString(key, sliceTable)
		default:
		}
	}
	return resultTable
}

// toGoValue converts the given LValue to a Go object.
func toGoValue(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LString:
		return string(v)
	case lua.LNumber:
		return float64(v)
	case *lua.LTable:
		maxn := v.MaxN()
		// Check for __is_array metatable to force array output
		if mt := v.Metatable; mt != nil {
			if arrFlag := mt.RawGetString("__is_array"); arrFlag != lua.LNil && arrFlag == lua.LTrue {
				// Always treat as array, even if empty
				ret := make([]interface{}, 0, maxn)
				for i := 1; i <= maxn; i++ {
					ret = append(ret, toGoValue(v.RawGetInt(i)))
				}
				return ret
			}
		}
		if maxn == 0 {
			ret := make(map[string]interface{})
			v.ForEach(func(key, value lua.LValue) {
				keystr := key.String()
				ret[keystr] = toGoValue(value)
			})
			return ret
		} else { // array
			ret := make([]interface{}, 0, maxn)
			for i := 1; i <= maxn; i++ {
				ret = append(ret, toGoValue(v.RawGetInt(i)))
			}
			return ret
		}
	default:
		return v
	}
}
