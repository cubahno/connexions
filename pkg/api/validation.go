package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/doordash-oss/oapi-codegen-dd/v3/pkg/runtime"
)

// ValidationStats tracks how often validation is performed
type ValidationStats struct {
	// Number of times Validate() was called
	WithValidator int64

	// Number of times type didn't implement Validator
	WithoutValidator int64

	// Number of times type didn't implement Validator
	Skipped int64
}

var validationStats ValidationStats

// GetValidationStats returns the current validation statistics
func GetValidationStats() ValidationStats {
	return ValidationStats{
		WithValidator:    atomic.LoadInt64(&validationStats.WithValidator),
		WithoutValidator: atomic.LoadInt64(&validationStats.WithoutValidator),
		Skipped:          atomic.LoadInt64(&validationStats.Skipped),
	}
}

// ResetValidationStats resets the validation statistics (useful for testing)
func ResetValidationStats() {
	atomic.StoreInt64(&validationStats.WithValidator, 0)
	atomic.StoreInt64(&validationStats.WithoutValidator, 0)
	atomic.StoreInt64(&validationStats.Skipped, 0)
}

func ValidateRequest[T any](r *http.Request) error {
	var body T
	var (
		validated    bool
		nonValidated bool
		skipped      bool
	)

	// Track validation stats with defer to ensure it's always called
	defer func() {
		if validated {
			atomic.AddInt64(&validationStats.WithValidator, 1)
		} else if nonValidated {
			atomic.AddInt64(&validationStats.WithoutValidator, 1)
		} else if skipped {
			atomic.AddInt64(&validationStats.Skipped, 1)
		}
	}()

	// Early return if no body
	if r.Body == nil {
		skipped = true
		return nil
	}

	// Read the body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Early return if empty body
	if len(bodyBytes) == 0 {
		skipped = true
		return nil
	}

	// Check Content-Type to determine how to decode the body
	contentType := r.Header.Get("Content-Type")
	isFormEncoded := strings.Contains(strings.ToLower(contentType), "application/x-www-form-urlencoded")

	if isFormEncoded {
		// Convert form-encoded data to JSON using runtime.ConvertFormFields
		// This handles deepObject encoding (e.g., "obj[key][nested]=value") and type conversion
		jsonBytes, err := runtime.ConvertFormFields(bodyBytes)
		if err != nil {
			return fmt.Errorf("failed to convert form data: %w", err)
		}

		// Parse to map to remove empty objects (for union type compatibility)
		var formData map[string]any
		if err := json.Unmarshal(jsonBytes, &formData); err != nil {
			return fmt.Errorf("failed to parse form data JSON: %w", err)
		}

		// Remove empty objects from form data to avoid union type unmarshaling errors
		// Empty objects don't match any variant of a union type (anyOf/oneOf)
		removeEmptyObjects(formData)

		// Re-marshal and unmarshal to target type using lenient decoder
		jsonBytes, err = json.Marshal(formData)
		if err != nil {
			return fmt.Errorf("failed to convert form data to JSON: %w", err)
		}

		// Use a custom unmarshaler that can handle string-to-number conversions
		if err := unmarshalLenient(jsonBytes, &body); err != nil {
			return fmt.Errorf("failed to unmarshal form data to target type: %w", err)
		}
	} else {
		// Unmarshal JSON body
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			return fmt.Errorf("failed to unmarshal request body: %w", err)
		}
	}

	if val, ok := any(body).(runtime.Validator); ok {
		validated = true
		// Use the generated Validate() method which properly handles nested validation
		// without checking struct-level required tags (which fail for zero-value structs)
		if err := val.Validate(); err != nil {
			return fmt.Errorf("request validation failed: %w", err)
		}
	} else {
		nonValidated = true
	}

	return nil
}

func ValidateResponse[T any](body []byte, contentType string) error {
	var response T
	var (
		validated    bool
		nonValidated bool
		skipped      bool
	)

	// Track validation stats with defer to ensure it's always called
	defer func() {
		if validated {
			atomic.AddInt64(&validationStats.WithValidator, 1)
		} else if nonValidated {
			atomic.AddInt64(&validationStats.WithoutValidator, 1)
		} else if skipped {
			atomic.AddInt64(&validationStats.Skipped, 1)
		}
	}()

	// Check for empty body
	if len(body) == 0 {
		return fmt.Errorf("generator produced empty response body")
	}

	// If validation is disabled or not JSON, return original body
	if contentType != "application/json" {
		skipped = true
		return nil
	}

	// Unmarshal the response
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("unmarshal failed: %w", err)
	}

	// Check if the type implements runtime.Validator interface for custom validation
	if val, ok := any(response).(runtime.Validator); ok {
		validated = true
		// Use the generated Validate() method which properly handles nested validation
		// without checking struct-level required tags (which fail for zero-value structs)
		if err := val.Validate(); err != nil {
			return fmt.Errorf("response validation failed: %w", err)
		}
	} else {
		nonValidated = true
	}

	return nil
}

// removeEmptyObjects recursively removes empty objects from a map.
// This is needed for form-encoded union types where empty objects don't match any variant.
func removeEmptyObjects(data map[string]any) {
	removeEmptyObjectsWithDepth(data, 0)
}

func removeEmptyObjectsWithDepth(data map[string]any, depth int) {
	// Prevent infinite recursion - form data shouldn't be deeper than 20 levels
	if depth > 20 {
		return
	}

	// Collect keys to delete after iteration to avoid modifying map during iteration
	var keysToDelete []string

	for key, value := range data {
		switch v := value.(type) {
		case map[string]any:
			if len(v) == 0 {
				// Mark for deletion
				keysToDelete = append(keysToDelete, key)
			} else {
				// Recursively clean nested objects
				removeEmptyObjectsWithDepth(v, depth+1)
				// Check again after cleaning - might be empty now
				if len(v) == 0 {
					keysToDelete = append(keysToDelete, key)
				}
			}
		case []any:
			// Clean objects inside arrays
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					removeEmptyObjectsWithDepth(m, depth+1)
				}
			}
		}
	}

	// Delete marked keys
	for _, key := range keysToDelete {
		delete(data, key)
	}
}

// unmarshalLenient unmarshals JSON data into the target with lenient type conversions.
// This is needed for form-encoded data where we can't distinguish between string enum values
// (e.g., "0", "1") and integer values without schema information.
// The standard json.Unmarshal is strict and won't convert numbers to strings or vice versa.
func unmarshalLenient(data []byte, target any) error {
	// First try standard JSON unmarshaling
	// If it succeeds, we're done (this handles most cases correctly)
	standardErr := json.Unmarshal(data, target)
	if standardErr == nil {
		return nil
	}

	// If standard unmarshaling fails, try lenient conversion
	// This handles cases like integer -> string for string enums
	var rawData map[string]any
	if err := json.Unmarshal(data, &rawData); err != nil {
		// If we can't even parse the JSON into a map, return the original error
		return standardErr
	}

	// Use reflection to set values with type conversion
	if err := setFieldsLenient(reflect.ValueOf(target).Elem(), rawData); err != nil {
		// If lenient conversion also fails, return both errors for better debugging
		return fmt.Errorf("standard unmarshal failed: %w; lenient unmarshal also failed: %v", standardErr, err)
	}

	return nil
}

// setFieldsLenient recursively sets struct fields from a map with lenient type conversions.
func setFieldsLenient(target reflect.Value, data map[string]any) error {
	targetType := target.Type()

	for i := 0; i < target.NumField(); i++ {
		field := target.Field(i)
		fieldType := targetType.Field(i)

		// Get the JSON tag name
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Handle "name,omitempty" format
		jsonName := strings.Split(jsonTag, ",")[0]

		// Get the value from the data map
		value, ok := data[jsonName]
		if !ok {
			continue
		}

		// Set the field with lenient type conversion
		if err := setValueLenient(field, value); err != nil {
			return fmt.Errorf("field %s: %w", jsonName, err)
		}
	}

	return nil
}

// setValueLenient sets a reflect.Value with lenient type conversion.
func setValueLenient(target reflect.Value, value any) error {
	if value == nil {
		return nil
	}

	targetType := target.Type()
	valueType := reflect.TypeOf(value)

	// Handle pointer types
	if targetType.Kind() == reflect.Ptr {
		if target.IsNil() {
			target.Set(reflect.New(targetType.Elem()))
		}
		return setValueLenient(target.Elem(), value)
	}

	// Direct assignment if types match
	if valueType.AssignableTo(targetType) {
		target.Set(reflect.ValueOf(value))
		return nil
	}

	// Handle type conversions
	switch targetType.Kind() {
	case reflect.String:
		// Convert numbers to strings (for string enums with numeric values)
		switch v := value.(type) {
		case float64:
			// Check if it's actually an integer
			if v == float64(int64(v)) {
				target.SetString(strconv.FormatInt(int64(v), 10))
			} else {
				target.SetString(strconv.FormatFloat(v, 'f', -1, 64))
			}
		case int64:
			target.SetString(strconv.FormatInt(v, 10))
		case string:
			target.SetString(v)
		case bool:
			target.SetString(strconv.FormatBool(v))
		default:
			return fmt.Errorf("cannot convert %T to string", value)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Convert strings to integers
		switch v := value.(type) {
		case float64:
			target.SetInt(int64(v))
		case int64:
			target.SetInt(v)
		case string:
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("cannot convert string %q to int: %w", v, err)
			}
			target.SetInt(i)
		default:
			return fmt.Errorf("cannot convert %T to int", value)
		}

	case reflect.Float32, reflect.Float64:
		// Convert strings to floats
		switch v := value.(type) {
		case float64:
			target.SetFloat(v)
		case int64:
			target.SetFloat(float64(v))
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return fmt.Errorf("cannot convert string %q to float: %w", v, err)
			}
			target.SetFloat(f)
		default:
			return fmt.Errorf("cannot convert %T to float", value)
		}

	case reflect.Bool:
		switch v := value.(type) {
		case bool:
			target.SetBool(v)
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("cannot convert string %q to bool: %w", v, err)
			}
			target.SetBool(b)
		default:
			return fmt.Errorf("cannot convert %T to bool", value)
		}

	case reflect.Struct:
		// Recursively handle nested structs
		if m, ok := value.(map[string]any); ok {
			return setFieldsLenient(target, m)
		}
		return fmt.Errorf("cannot convert %T to struct", value)

	case reflect.Slice:
		// Handle slices
		if arr, ok := value.([]any); ok {
			slice := reflect.MakeSlice(targetType, len(arr), len(arr))
			for i, item := range arr {
				if err := setValueLenient(slice.Index(i), item); err != nil {
					return fmt.Errorf("slice index %d: %w", i, err)
				}
			}
			target.Set(slice)
			return nil
		}
		return fmt.Errorf("cannot convert %T to slice", value)

	case reflect.Map:
		// Handle maps
		if m, ok := value.(map[string]any); ok {
			mapValue := reflect.MakeMap(targetType)
			for k, v := range m {
				keyValue := reflect.ValueOf(k)
				elemValue := reflect.New(targetType.Elem()).Elem()
				if err := setValueLenient(elemValue, v); err != nil {
					return fmt.Errorf("map key %s: %w", k, err)
				}
				mapValue.SetMapIndex(keyValue, elemValue)
			}
			target.Set(mapValue)
			return nil
		}
		return fmt.Errorf("cannot convert %T to map", value)

	default:
		return fmt.Errorf("unsupported target type: %s", targetType.Kind())
	}

	return nil
}
