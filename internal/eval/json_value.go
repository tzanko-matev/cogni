package eval

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// JSONKind identifies the concrete type stored in a JSONValue.
type JSONKind int

const (
	JSONNull JSONKind = iota
	JSONString
	JSONNumber
	JSONBool
	JSONObject
	JSONArray
)

// JSONValue represents an arbitrary JSON value without using empty interfaces.
type JSONValue struct {
	Kind   JSONKind
	String string
	Number float64
	Bool   bool
	Object map[string]JSONValue
	Array  []JSONValue
}

// UnmarshalJSON decodes a JSON value into the typed JSONValue representation.
func (v *JSONValue) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return fmt.Errorf("empty json value")
	}
	switch trimmed[0] {
	case '{':
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &raw); err != nil {
			return err
		}
		v.Kind = JSONObject
		v.Object = make(map[string]JSONValue, len(raw))
		for key, value := range raw {
			var child JSONValue
			if err := json.Unmarshal(value, &child); err != nil {
				return err
			}
			v.Object[key] = child
		}
		return nil
	case '[':
		var raw []json.RawMessage
		if err := json.Unmarshal(trimmed, &raw); err != nil {
			return err
		}
		v.Kind = JSONArray
		v.Array = make([]JSONValue, 0, len(raw))
		for _, value := range raw {
			var child JSONValue
			if err := json.Unmarshal(value, &child); err != nil {
				return err
			}
			v.Array = append(v.Array, child)
		}
		return nil
	case '"':
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return err
		}
		v.Kind = JSONString
		v.String = value
		return nil
	case 't', 'f':
		var value bool
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return err
		}
		v.Kind = JSONBool
		v.Bool = value
		return nil
	case 'n':
		if string(trimmed) != "null" {
			return fmt.Errorf("invalid json literal")
		}
		v.Kind = JSONNull
		return nil
	default:
		var value float64
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return err
		}
		v.Kind = JSONNumber
		v.Number = value
		return nil
	}
}

// ObjectValue returns the object map when the value is an object.
func (v JSONValue) ObjectValue() (map[string]JSONValue, bool) {
	if v.Kind != JSONObject {
		return nil, false
	}
	return v.Object, true
}

// ArrayValue returns the array slice when the value is an array.
func (v JSONValue) ArrayValue() ([]JSONValue, bool) {
	if v.Kind != JSONArray {
		return nil, false
	}
	return v.Array, true
}

// StringValue returns the string when the value is a string.
func (v JSONValue) StringValue() (string, bool) {
	if v.Kind != JSONString {
		return "", false
	}
	return v.String, true
}

// NumberValue returns the number when the value is numeric.
func (v JSONValue) NumberValue() (float64, bool) {
	if v.Kind != JSONNumber {
		return 0, false
	}
	return v.Number, true
}

// BoolValue returns the boolean when the value is a bool.
func (v JSONValue) BoolValue() (bool, bool) {
	if v.Kind != JSONBool {
		return false, false
	}
	return v.Bool, true
}

// ToInterface converts the JSONValue into standard Go JSON types.
func (v JSONValue) ToInterface() interface{} {
	switch v.Kind {
	case JSONObject:
		out := make(map[string]interface{}, len(v.Object))
		for key, value := range v.Object {
			out[key] = value.ToInterface()
		}
		return out
	case JSONArray:
		out := make([]interface{}, 0, len(v.Array))
		for _, value := range v.Array {
			out = append(out, value.ToInterface())
		}
		return out
	case JSONString:
		return v.String
	case JSONNumber:
		return v.Number
	case JSONBool:
		return v.Bool
	case JSONNull:
		return nil
	default:
		return nil
	}
}
