package dependencies

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/itchyny/gojq"
	"github.com/tidwall/gjson"
)

func Transform(event *cloudevents.Event, expr string) (*cloudevents.Event, error) {
	resultEvent := event.Clone()

	payload := event.Data()

	if payload != nil && !gjson.Valid(string(payload)) {
		return nil, fmt.Errorf("event payload is not a valid json object")
	}

	var queryInput map[string]interface{}
	if err := json.Unmarshal(payload, &queryInput); err != nil {
		return nil, fmt.Errorf("failed to parse event payload. err: %s", err.Error())
	}

	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, err
	}

	var result interface{}
	iter := query.Run(queryInput)
	for {
		tmpResult, ok := iter.Next()
		if !ok {
			break
		}
		result = tmpResult
		if err, ok = result.(error); ok {
			return nil, fmt.Errorf("failed to parse run jq expression %s. err: %s", expr, err.Error())
		}
	}

	output, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse jq output json. err: %s", err.Error())
	}

	if output != nil && !gjson.Valid(string(output)) {
		return nil, fmt.Errorf("jq output is not a valid json object: %s", string(output))
	}

	if err = resultEvent.SetData(cloudevents.ApplicationJSON, output); err != nil {
		return nil, err
	}
	return &resultEvent, nil
}
