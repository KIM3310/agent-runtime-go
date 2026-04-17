package runtime

import (
	"fmt"
)

// validateArgs is a minimal JSON Schema validator.
// Handles the subset of JSON Schema relevant for tool arguments:
// type, required, properties, enum, items, additionalProperties.
//
// For full JSON Schema support, plug in a library like xeipuuv/gojsonschema.
// This minimal validator keeps dependencies low.
func validateArgs(args map[string]any, schema map[string]any) error {
	schemaType, _ := schema["type"].(string)
	if schemaType != "object" {
		return fmt.Errorf("root schema must be object, got %q", schemaType)
	}

	// Check required fields
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			reqKey, _ := r.(string)
			if _, present := args[reqKey]; !present {
				return fmt.Errorf("required field missing: %s", reqKey)
			}
		}
	} else if req, ok := schema["required"].([]string); ok {
		for _, r := range req {
			if _, present := args[r]; !present {
				return fmt.Errorf("required field missing: %s", r)
			}
		}
	}

	// Check each property's type
	if props, ok := schema["properties"].(map[string]any); ok {
		for key, val := range args {
			propSchema, ok := props[key].(map[string]any)
			if !ok {
				// additionalProperties check
				if addProps, ok := schema["additionalProperties"].(bool); ok && !addProps {
					return fmt.Errorf("additional property not allowed: %s", key)
				}
				continue
			}
			if err := validateType(key, val, propSchema); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateType(key string, val any, schema map[string]any) error {
	schemaType, _ := schema["type"].(string)

	switch schemaType {
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Errorf("field %s: expected string, got %T", key, val)
		}
		// enum check
		if enum, ok := schema["enum"].([]any); ok {
			valStr, _ := val.(string)
			matched := false
			for _, e := range enum {
				if eStr, _ := e.(string); eStr == valStr {
					matched = true
					break
				}
			}
			if !matched {
				return fmt.Errorf("field %s: value %q not in enum", key, valStr)
			}
		}
	case "number", "integer":
		switch val.(type) {
		case float64, float32, int, int32, int64:
			// OK
		default:
			return fmt.Errorf("field %s: expected number, got %T", key, val)
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("field %s: expected boolean, got %T", key, val)
		}
	case "array":
		arr, ok := val.([]any)
		if !ok {
			return fmt.Errorf("field %s: expected array, got %T", key, val)
		}
		if itemSchema, ok := schema["items"].(map[string]any); ok {
			for i, item := range arr {
				if err := validateType(fmt.Sprintf("%s[%d]", key, i), item, itemSchema); err != nil {
					return err
				}
			}
		}
	case "object":
		sub, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("field %s: expected object, got %T", key, val)
		}
		return validateArgs(sub, schema)
	case "":
		// no type specified; accept anything
	default:
		return fmt.Errorf("field %s: unsupported schema type %q", key, schemaType)
	}

	return nil
}
