package goservice

import (
	"fmt"
	"strings"
)

// ParamRule defines the validation rules for a single action parameter.
type ParamRule struct {
	// Type is the expected parameter type: "string", "number", "bool", "array", "object", or "any".
	Type string
	// Required indicates that the parameter must be present and non-nil.
	Required bool
	// Min is the minimum value for numbers or minimum length for strings (optional).
	Min *float64
	// Max is the maximum value for numbers or maximum length for strings (optional).
	Max *float64
}

// FieldError represents a validation failure for a single parameter field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationError is returned when one or more parameter validation rules fail.
type ValidationError struct {
	Fields []FieldError `json:"fields"`
}

func (e *ValidationError) Error() string {
	msgs := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		msgs[i] = f.Field + ": " + f.Message
	}
	return "Validation error: " + strings.Join(msgs, "; ")
}

// validateParams checks params against the provided schema.
// It returns nil when all rules pass or a *ValidationError describing every failing field.
func validateParams(params interface{}, schema map[string]ParamRule) error {
	var fieldErrors []FieldError

	paramsMap, isMap := params.(map[string]interface{})
	if !isMap {
		// params is not a map; only presence-checking is possible
		for field, rule := range schema {
			if rule.Required {
				fieldErrors = append(fieldErrors, FieldError{Field: field, Message: "is required"})
			}
		}
		if len(fieldErrors) > 0 {
			return &ValidationError{Fields: fieldErrors}
		}
		return nil
	}

	for field, rule := range schema {
		val, exists := paramsMap[field]
		if !exists || val == nil {
			if rule.Required {
				fieldErrors = append(fieldErrors, FieldError{Field: field, Message: "is required"})
			}
			continue
		}

		switch rule.Type {
		case "string":
			s, ok := val.(string)
			if !ok {
				fieldErrors = append(fieldErrors, FieldError{Field: field, Message: "must be a string"})
				continue
			}
			length := float64(len(s))
			if rule.Min != nil && length < *rule.Min {
				fieldErrors = append(fieldErrors, FieldError{Field: field, Message: fmt.Sprintf("length must be at least %.0f", *rule.Min)})
			}
			if rule.Max != nil && length > *rule.Max {
				fieldErrors = append(fieldErrors, FieldError{Field: field, Message: fmt.Sprintf("length must be at most %.0f", *rule.Max)})
			}

		case "number":
			var num float64
			switch v := val.(type) {
			case float64:
				num = v
			case float32:
				num = float64(v)
			case int:
				num = float64(v)
			case int32:
				num = float64(v)
			case int64:
				num = float64(v)
			default:
				fieldErrors = append(fieldErrors, FieldError{Field: field, Message: "must be a number"})
				continue
			}
			if rule.Min != nil && num < *rule.Min {
				fieldErrors = append(fieldErrors, FieldError{Field: field, Message: fmt.Sprintf("must be at least %g", *rule.Min)})
			}
			if rule.Max != nil && num > *rule.Max {
				fieldErrors = append(fieldErrors, FieldError{Field: field, Message: fmt.Sprintf("must be at most %g", *rule.Max)})
			}

		case "bool":
			if _, ok := val.(bool); !ok {
				fieldErrors = append(fieldErrors, FieldError{Field: field, Message: "must be a boolean"})
			}

		case "array":
			if _, ok := val.([]interface{}); !ok {
				fieldErrors = append(fieldErrors, FieldError{Field: field, Message: "must be an array"})
			}

		case "object":
			if _, ok := val.(map[string]interface{}); !ok {
				fieldErrors = append(fieldErrors, FieldError{Field: field, Message: "must be an object"})
			}

		// "any" or empty type: accept all values
		}
	}

	if len(fieldErrors) > 0 {
		return &ValidationError{Fields: fieldErrors}
	}
	return nil
}
