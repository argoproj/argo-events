package v1alpha1

import (
	"encoding/json"
	"reflect"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestFromInt64(t *testing.T) {
	i := FromInt64(93)
	if i.Type != Int64 || i.Int64Val != 93 {
		t.Errorf("Expected Int64Val=93, got %+v", i)
	}
}

func TestFromString(t *testing.T) {
	i := FromString("76")
	if i.Type != String || i.StrVal != "76" {
		t.Errorf("Expected StrVal=\"76\", got %+v", i)
	}
}

type Int64OrStringHolder struct {
	IOrS Int64OrString `json:"val"`
}

func TestInt64OrStringUnmarshalJSON(t *testing.T) {
	cases := []struct {
		input  string
		result Int64OrString
	}{
		{"{\"val\": 123}", FromInt64(123)},
		{"{\"val\": \"123\"}", FromString("123")},
	}

	for _, c := range cases {
		var result Int64OrStringHolder
		if err := json.Unmarshal([]byte(c.input), &result); err != nil {
			t.Errorf("Failed to unmarshal input '%v': %v", c.input, err)
		}
		if result.IOrS != c.result {
			t.Errorf("Failed to unmarshal input '%v': expected %+v, got %+v", c.input, c.result, result)
		}
	}
}

func TestInt64OrStringMarshalJSON(t *testing.T) {
	cases := []struct {
		input  Int64OrString
		result string
	}{
		{FromInt64(123), "{\"val\":123}"},
		{FromString("123"), "{\"val\":\"123\"}"},
	}

	for _, c := range cases {
		input := Int64OrStringHolder{c.input}
		result, err := json.Marshal(&input)
		if err != nil {
			t.Errorf("Failed to marshal input '%v': %v", input, err)
		}
		if string(result) != c.result {
			t.Errorf("Failed to marshal input '%v': expected: %+v, got %q", input, c.result, string(result))
		}
	}
}

func TestInt64OrStringMarshalJSONUnmarshalYAML(t *testing.T) {
	cases := []struct {
		input Int64OrString
	}{
		{FromInt64(123)},
		{FromString("123")},
	}

	for _, c := range cases {
		input := Int64OrStringHolder{c.input}
		jsonMarshalled, err := json.Marshal(&input)
		if err != nil {
			t.Errorf("1: Failed to marshal input: '%v': %v", input, err)
		}

		var result Int64OrStringHolder
		err = yaml.Unmarshal(jsonMarshalled, &result)
		if err != nil {
			t.Errorf("2: Failed to unmarshal '%+v': %v", string(jsonMarshalled), err)
		}

		if !reflect.DeepEqual(input, result) {
			t.Errorf("3: Failed to marshal input '%+v': got %+v", input, result)
		}
	}
}
