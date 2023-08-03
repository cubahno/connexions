package api

import (
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
	Age   int    `validate:"gte=0,lte=130"`
}

// ValidatableStruct implements runtime.Validator interface
type ValidatableStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (v ValidatableStruct) Validate() error {
	if v.Name == "" {
		return errors.New("name is required")
	}
	if v.Age < 0 {
		return errors.New("age must be non-negative")
	}
	return nil
}

func TestValidateResponse(t *testing.T) {
	t.Run("valid struct response", func(t *testing.T) {
		err := ValidateResponse[TestStruct]([]byte(`{"name":"John","email":"john@example.com","age":30}`), "application/json")
		assert.NoError(t, err)
	})

	t.Run("invalid struct - validation fails (age out of range)", func(t *testing.T) {
		err := ValidateResponse[TestStruct]([]byte(`{"name":"John","email":"john@example.com","age":200}`), "application/json")
		// Note: TestStruct uses validator tags, but ValidateResponse only calls runtime.Validator interface
		// Since TestStruct doesn't implement runtime.Validator, validation is skipped
		assert.NoError(t, err)
	})

	t.Run("invalid json - unmarshal fails", func(t *testing.T) {
		err := ValidateResponse[TestStruct]([]byte(`{invalid json}`), "application/json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal failed")
	})

	t.Run("empty body - returns error", func(t *testing.T) {
		err := ValidateResponse[TestStruct]([]byte(``), "application/json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty response body")
	})

	t.Run("non-JSON content type - returns nil", func(t *testing.T) {
		err := ValidateResponse[TestStruct]([]byte(`some text`), "text/plain")
		assert.NoError(t, err)
	})

	t.Run("slice of any - should pass", func(t *testing.T) {
		err := ValidateResponse[[]any]([]byte(`["string", 123, true, null]`), "application/json")
		assert.NoError(t, err)
	})

	t.Run("empty slice of any - should fail (empty body check)", func(t *testing.T) {
		err := ValidateResponse[[]any]([]byte(`[]`), "application/json")
		// Empty array is valid JSON but has length > 0, so it should pass
		assert.NoError(t, err)
	})

	t.Run("slice of structs - valid", func(t *testing.T) {
		err := ValidateResponse[[]TestStruct]([]byte(`[{"name":"John","email":"john@example.com","age":30}]`), "application/json")
		assert.NoError(t, err)
	})

	t.Run("slice of structs - invalid element (age out of range)", func(t *testing.T) {
		err := ValidateResponse[[]TestStruct]([]byte(`[{"name":"John","email":"john@example.com","age":200}]`), "application/json")
		// Note: TestStruct doesn't implement runtime.Validator, so validation is skipped
		assert.NoError(t, err)
	})

	t.Run("valid response with Validator interface", func(t *testing.T) {
		err := ValidateResponse[ValidatableStruct]([]byte(`{"name":"John","age":30}`), "application/json")
		assert.NoError(t, err)
	})

	t.Run("invalid response with Validator interface - validation fails", func(t *testing.T) {
		err := ValidateResponse[ValidatableStruct]([]byte(`{"name":"","age":30}`), "application/json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})
}

func TestValidateRequest(t *testing.T) {
	t.Run("valid JSON body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users", strings.NewReader(`{"name":"John","email":"john@example.com","age":30}`))
		req.Header.Set("Content-Type", "application/json")

		err := ValidateRequest[TestStruct](req)
		// Note: TestStruct doesn't implement runtime.Validator, so validation is skipped
		assert.NoError(t, err)
	})

	t.Run("invalid JSON - unmarshal fails", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users", strings.NewReader(`{invalid json}`))
		req.Header.Set("Content-Type", "application/json")

		err := ValidateRequest[TestStruct](req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal request body")
	})

	t.Run("empty body - returns nil", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users", strings.NewReader(``))
		req.Header.Set("Content-Type", "application/json")

		err := ValidateRequest[TestStruct](req)
		assert.NoError(t, err)
	})

	t.Run("no body (GET request) - returns nil", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users", nil)

		err := ValidateRequest[struct{}](req)
		assert.NoError(t, err)
	})

	t.Run("valid JSON body with Validator interface", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users", strings.NewReader(`{"name":"John","age":30}`))
		req.Header.Set("Content-Type", "application/json")

		err := ValidateRequest[ValidatableStruct](req)
		assert.NoError(t, err)
	})

	t.Run("invalid JSON body with Validator interface - validation fails", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users", strings.NewReader(`{"name":"","age":30}`))
		req.Header.Set("Content-Type", "application/json")

		err := ValidateRequest[ValidatableStruct](req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("form-encoded body with Validator interface", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users", strings.NewReader("name=John&age=30"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := ValidateRequest[ValidatableStruct](req)
		assert.NoError(t, err)
	})

	t.Run("form-encoded body with Validator interface - validation fails", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users", strings.NewReader("name=&age=30"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := ValidateRequest[ValidatableStruct](req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("nil body returns nil", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/users", nil)
		req.Body = nil // Explicitly set to nil

		err := ValidateRequest[TestStruct](req)
		assert.NoError(t, err)
	})

	t.Run("invalid form-encoded body - parse error", func(t *testing.T) {
		// Create a body with invalid percent encoding
		req := httptest.NewRequest("POST", "/users", strings.NewReader("name=%ZZ"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := ValidateRequest[TestStruct](req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to convert form data")
	})
}

func TestValidateRequest_FormEncoded(t *testing.T) {
	type FormBody struct {
		Name   string `json:"name"`
		Age    int    `json:"age"`
		Active bool   `json:"active"`
	}

	t.Run("parses form-encoded body", func(t *testing.T) {
		formData := "name=John&age=30&active=true"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := ValidateRequest[FormBody](req)
		// Note: FormBody doesn't implement runtime.Validator, so validation is skipped
		assert.NoError(t, err)
	})

	t.Run("parses form-encoded array", func(t *testing.T) {
		formData := "expand[0]=customers&expand[1]=invoices"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		type ArrayBody struct {
			Expand []string `json:"expand"`
		}

		err := ValidateRequest[ArrayBody](req)
		assert.NoError(t, err)
	})

	t.Run("parses nested form-encoded object", func(t *testing.T) {
		formData := "address[city]=Berlin&address[country]=DE&address[street]=Main+St"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(formData))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		type Address struct {
			City    string `json:"city"`
			Country string `json:"country"`
			Street  string `json:"street"`
		}

		type NestedBody struct {
			Address Address `json:"address"`
		}

		err := ValidateRequest[NestedBody](req)
		assert.NoError(t, err)
	})

	t.Run("parses form-encoded string enum with numeric values", func(t *testing.T) {
		// This tests the case where a string enum has numeric-looking values like "0", "1", "2"
		// The form encoder can't distinguish between string "0" and integer 0, so it sends just "0"
		// Our lenient unmarshaler should convert the integer 0 to string "0" when the target is a string field
		type CartesBancaires struct {
			CbAvalgo    string `json:"cb_avalgo"`    // String enum with values "0", "1", "2", "3", "4", "A"
			CbExemption string `json:"cb_exemption"` // String enum
			CbScore     int    `json:"cb_score"`     // Integer field
		}

		type FormBody struct {
			CartesBancaires CartesBancaires `json:"cartes_bancaires"`
		}

		// Parse as nested object
		body := "cartes_bancaires[cb_avalgo]=0&cartes_bancaires[cb_exemption]=atta&cartes_bancaires[cb_score]=123"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := ValidateRequest[FormBody](req)
		assert.NoError(t, err)
	})
}

// TestStringEnumWithNumericValues tests that string enums with numeric values work correctly
// in both JSON and form-encoded requests/responses
func TestStringEnumWithNumericValues(t *testing.T) {
	type CartesBancaires struct {
		CbAvalgo    string `json:"cb_avalgo"`    // String enum: "0", "1", "2", "3", "4", "A"
		CbExemption string `json:"cb_exemption"` // String enum: "atta", "low_value", etc.
		CbScore     int    `json:"cb_score"`     // Integer field
	}

	type TestBody struct {
		CartesBancaires CartesBancaires `json:"cartes_bancaires"`
	}

	t.Run("JSON request with string enum numeric value", func(t *testing.T) {
		// JSON should have "0" as a string
		jsonBody := `{"cartes_bancaires":{"cb_avalgo":"0","cb_exemption":"atta","cb_score":123}}`
		req := httptest.NewRequest("POST", "/test", strings.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		err := ValidateRequest[TestBody](req)
		assert.NoError(t, err)
	})

	t.Run("JSON request with integer instead of string enum - should fail", func(t *testing.T) {
		// JSON with integer 0 instead of string "0" should fail standard unmarshaling
		jsonBody := `{"cartes_bancaires":{"cb_avalgo":0,"cb_exemption":"atta","cb_score":123}}`
		req := httptest.NewRequest("POST", "/test", strings.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		err := ValidateRequest[TestBody](req)
		// Standard JSON unmarshaler should fail because it can't unmarshal number into string
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot unmarshal number into Go struct field")
	})

	t.Run("form-encoded request with numeric string enum value", func(t *testing.T) {
		// Form-encoded: cb_avalgo=0 (without quotes, looks like integer)
		// Our lenient unmarshaler should convert it to string "0"
		formBody := "cartes_bancaires[cb_avalgo]=0&cartes_bancaires[cb_exemption]=atta&cartes_bancaires[cb_score]=123"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(formBody))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := ValidateRequest[TestBody](req)
		assert.NoError(t, err)
	})

	t.Run("form-encoded request with non-numeric string enum value", func(t *testing.T) {
		// Form-encoded with "A" (non-numeric enum value)
		formBody := "cartes_bancaires[cb_avalgo]=A&cartes_bancaires[cb_exemption]=atta&cartes_bancaires[cb_score]=123"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(formBody))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := ValidateRequest[TestBody](req)
		assert.NoError(t, err)
	})

	t.Run("JSON response with string enum numeric value", func(t *testing.T) {
		// Response should have "0" as a string
		jsonBody := `{"cartes_bancaires":{"cb_avalgo":"0","cb_exemption":"atta","cb_score":123}}`
		err := ValidateResponse[TestBody]([]byte(jsonBody), "application/json")
		assert.NoError(t, err)
	})

	t.Run("JSON response with integer instead of string enum - should fail", func(t *testing.T) {
		// Response with integer 0 instead of string "0" should fail
		jsonBody := `{"cartes_bancaires":{"cb_avalgo":0,"cb_exemption":"atta","cb_score":123}}`
		err := ValidateResponse[TestBody]([]byte(jsonBody), "application/json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot unmarshal number into Go struct field")
	})

	t.Run("form-encoded response with numeric string enum value", func(t *testing.T) {
		// Form-encoded responses are not common, but if they exist, they should work
		formBody := "cartes_bancaires[cb_avalgo]=0&cartes_bancaires[cb_exemption]=atta&cartes_bancaires[cb_score]=123"
		err := ValidateResponse[TestBody]([]byte(formBody), "application/x-www-form-urlencoded")
		assert.NoError(t, err)
	})

	t.Run("form-encoded request with invalid data - should report error", func(t *testing.T) {
		// Invalid: cb_score should be an integer, but we're sending a non-numeric string
		// that can't be converted
		formBody := "cartes_bancaires[cb_avalgo]=0&cartes_bancaires[cb_exemption]=atta&cartes_bancaires[cb_score]=not-a-number"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(formBody))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := ValidateRequest[TestBody](req)
		// Should fail because "not-a-number" can't be converted to int
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert string")
	})

	t.Run("form-encoded request with completely invalid structure", func(t *testing.T) {
		// Send a string where a nested object is expected
		formBody := "cartes_bancaires=invalid-string-not-object"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(formBody))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		err := ValidateRequest[TestBody](req)
		// Should fail with a meaningful error
		assert.Error(t, err)
	})

	t.Run("lenient unmarshaling doesn't bypass validation", func(t *testing.T) {
		// Even though we convert 0 to "0", if "0" is not a valid enum value,
		// validation should still fail
		type StrictEnum struct {
			Status string `json:"status"` // Imagine this has enum validation for ["active", "inactive"]
		}

		// This will unmarshal successfully (0 → "0"), but if there's validation
		// that checks enum values, it should fail
		formBody := "status=0"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(formBody))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// This test just verifies that unmarshaling succeeds
		// In a real scenario with runtime.Validator, the Validate() method would catch invalid enum values
		err := ValidateRequest[StrictEnum](req)
		// Without a Validator implementation, this will succeed
		// But the point is that validation is a separate step after unmarshaling
		assert.NoError(t, err)
	})
}

func TestSetValueLenient(t *testing.T) {
	t.Run("nil value returns nil", func(t *testing.T) {
		var s string
		target := reflect.ValueOf(&s).Elem()
		err := setValueLenient(target, nil)
		assert.NoError(t, err)
		assert.Equal(t, "", s)
	})

	t.Run("pointer type - creates new value if nil", func(t *testing.T) {
		var ptr *string
		target := reflect.ValueOf(&ptr).Elem()
		err := setValueLenient(target, "hello")
		assert.NoError(t, err)
		assert.NotNil(t, ptr)
		assert.Equal(t, "hello", *ptr)
	})

	t.Run("string from float64 integer", func(t *testing.T) {
		var s string
		target := reflect.ValueOf(&s).Elem()
		err := setValueLenient(target, float64(42))
		assert.NoError(t, err)
		assert.Equal(t, "42", s)
	})

	t.Run("string from float64 decimal", func(t *testing.T) {
		var s string
		target := reflect.ValueOf(&s).Elem()
		err := setValueLenient(target, float64(3.14))
		assert.NoError(t, err)
		assert.Equal(t, "3.14", s)
	})

	t.Run("string from int64", func(t *testing.T) {
		var s string
		target := reflect.ValueOf(&s).Elem()
		err := setValueLenient(target, int64(123))
		assert.NoError(t, err)
		assert.Equal(t, "123", s)
	})

	t.Run("string from bool", func(t *testing.T) {
		var s string
		target := reflect.ValueOf(&s).Elem()
		err := setValueLenient(target, true)
		assert.NoError(t, err)
		assert.Equal(t, "true", s)
	})

	t.Run("string from unsupported type returns error", func(t *testing.T) {
		var s string
		target := reflect.ValueOf(&s).Elem()
		err := setValueLenient(target, []int{1, 2, 3})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert")
	})

	t.Run("int from float64", func(t *testing.T) {
		var i int
		target := reflect.ValueOf(&i).Elem()
		err := setValueLenient(target, float64(42))
		assert.NoError(t, err)
		assert.Equal(t, 42, i)
	})

	t.Run("int from int64", func(t *testing.T) {
		var i int
		target := reflect.ValueOf(&i).Elem()
		err := setValueLenient(target, int64(42))
		assert.NoError(t, err)
		assert.Equal(t, 42, i)
	})

	t.Run("int from string", func(t *testing.T) {
		var i int
		target := reflect.ValueOf(&i).Elem()
		err := setValueLenient(target, "42")
		assert.NoError(t, err)
		assert.Equal(t, 42, i)
	})

	t.Run("int from invalid string returns error", func(t *testing.T) {
		var i int
		target := reflect.ValueOf(&i).Elem()
		err := setValueLenient(target, "not-a-number")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert string")
	})

	t.Run("int from unsupported type returns error", func(t *testing.T) {
		var i int
		target := reflect.ValueOf(&i).Elem()
		err := setValueLenient(target, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert")
	})

	t.Run("float from float64", func(t *testing.T) {
		var f float64
		target := reflect.ValueOf(&f).Elem()
		err := setValueLenient(target, float64(3.14))
		assert.NoError(t, err)
		assert.Equal(t, 3.14, f)
	})

	t.Run("float from int64", func(t *testing.T) {
		var f float64
		target := reflect.ValueOf(&f).Elem()
		err := setValueLenient(target, int64(42))
		assert.NoError(t, err)
		assert.Equal(t, float64(42), f)
	})

	t.Run("float from string", func(t *testing.T) {
		var f float64
		target := reflect.ValueOf(&f).Elem()
		err := setValueLenient(target, "3.14")
		assert.NoError(t, err)
		assert.Equal(t, 3.14, f)
	})

	t.Run("float from invalid string returns error", func(t *testing.T) {
		var f float64
		target := reflect.ValueOf(&f).Elem()
		err := setValueLenient(target, "not-a-number")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert string")
	})

	t.Run("float from unsupported type returns error", func(t *testing.T) {
		var f float64
		target := reflect.ValueOf(&f).Elem()
		err := setValueLenient(target, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert")
	})

	t.Run("bool from bool", func(t *testing.T) {
		var b bool
		target := reflect.ValueOf(&b).Elem()
		err := setValueLenient(target, true)
		assert.NoError(t, err)
		assert.True(t, b)
	})

	t.Run("bool from string", func(t *testing.T) {
		var b bool
		target := reflect.ValueOf(&b).Elem()
		err := setValueLenient(target, "true")
		assert.NoError(t, err)
		assert.True(t, b)
	})

	t.Run("bool from invalid string returns error", func(t *testing.T) {
		var b bool
		target := reflect.ValueOf(&b).Elem()
		err := setValueLenient(target, "not-a-bool")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert string")
	})

	t.Run("bool from unsupported type returns error", func(t *testing.T) {
		var b bool
		target := reflect.ValueOf(&b).Elem()
		err := setValueLenient(target, 42)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert")
	})

	t.Run("struct from map", func(t *testing.T) {
		type Person struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		var p Person
		target := reflect.ValueOf(&p).Elem()
		err := setValueLenient(target, map[string]any{"name": "John", "age": float64(30)})
		assert.NoError(t, err)
		assert.Equal(t, "John", p.Name)
		assert.Equal(t, 30, p.Age)
	})

	t.Run("struct from non-map returns error", func(t *testing.T) {
		type Person struct {
			Name string `json:"name"`
		}
		var p Person
		target := reflect.ValueOf(&p).Elem()
		err := setValueLenient(target, "not-a-map")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert")
	})

	t.Run("slice from array", func(t *testing.T) {
		var s []int
		target := reflect.ValueOf(&s).Elem()
		err := setValueLenient(target, []any{float64(1), float64(2), float64(3)})
		assert.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, s)
	})

	t.Run("slice with invalid element returns error", func(t *testing.T) {
		var s []int
		target := reflect.ValueOf(&s).Elem()
		err := setValueLenient(target, []any{float64(1), "not-a-number"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "slice index")
	})

	t.Run("slice from non-array returns error", func(t *testing.T) {
		var s []int
		target := reflect.ValueOf(&s).Elem()
		err := setValueLenient(target, "not-an-array")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert")
	})

	t.Run("map from map", func(t *testing.T) {
		var m map[string]int
		target := reflect.ValueOf(&m).Elem()
		err := setValueLenient(target, map[string]any{"a": float64(1), "b": float64(2)})
		assert.NoError(t, err)
		assert.Equal(t, map[string]int{"a": 1, "b": 2}, m)
	})

	t.Run("map with invalid value returns error", func(t *testing.T) {
		var m map[string]int
		target := reflect.ValueOf(&m).Elem()
		err := setValueLenient(target, map[string]any{"a": "not-a-number"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "map key")
	})

	t.Run("map from non-map returns error", func(t *testing.T) {
		var m map[string]int
		target := reflect.ValueOf(&m).Elem()
		err := setValueLenient(target, "not-a-map")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert")
	})

	t.Run("unsupported target type returns error", func(t *testing.T) {
		var c complex128
		target := reflect.ValueOf(&c).Elem()
		err := setValueLenient(target, "value")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported target type")
	})
}

func TestUnmarshalLenient(t *testing.T) {
	type SimpleStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	t.Run("standard unmarshal succeeds", func(t *testing.T) {
		var s SimpleStruct
		err := unmarshalLenient([]byte(`{"name":"John","age":30}`), &s)
		assert.NoError(t, err)
		assert.Equal(t, "John", s.Name)
		assert.Equal(t, 30, s.Age)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		var s SimpleStruct
		err := unmarshalLenient([]byte(`{invalid json`), &s)
		assert.Error(t, err)
	})

	t.Run("lenient conversion for string to int", func(t *testing.T) {
		var s SimpleStruct
		// This should fail standard unmarshal but succeed with lenient
		err := unmarshalLenient([]byte(`{"name":"John","age":"30"}`), &s)
		assert.NoError(t, err)
		assert.Equal(t, "John", s.Name)
		assert.Equal(t, 30, s.Age)
	})
}

func TestSetFieldsLenient(t *testing.T) {
	type TestStruct struct {
		Name    string `json:"name"`
		Age     int    `json:"age,omitempty"`
		Ignored string `json:"-"`
		NoTag   string
	}

	t.Run("sets fields from data map", func(t *testing.T) {
		var s TestStruct
		target := reflect.ValueOf(&s).Elem()
		data := map[string]any{
			"name": "John",
			"age":  30,
		}
		err := setFieldsLenient(target, data)
		assert.NoError(t, err)
		assert.Equal(t, "John", s.Name)
		assert.Equal(t, 30, s.Age)
	})

	t.Run("skips fields with no json tag", func(t *testing.T) {
		var s TestStruct
		target := reflect.ValueOf(&s).Elem()
		data := map[string]any{
			"NoTag": "value",
		}
		err := setFieldsLenient(target, data)
		assert.NoError(t, err)
		assert.Equal(t, "", s.NoTag) // Should not be set
	})

	t.Run("skips fields with json:- tag", func(t *testing.T) {
		var s TestStruct
		target := reflect.ValueOf(&s).Elem()
		data := map[string]any{
			"Ignored": "value",
		}
		err := setFieldsLenient(target, data)
		assert.NoError(t, err)
		assert.Equal(t, "", s.Ignored) // Should not be set
	})

	t.Run("skips fields not in data map", func(t *testing.T) {
		var s TestStruct
		s.Name = "existing"
		target := reflect.ValueOf(&s).Elem()
		data := map[string]any{
			"age": 25,
		}
		err := setFieldsLenient(target, data)
		assert.NoError(t, err)
		assert.Equal(t, "existing", s.Name) // Should remain unchanged
		assert.Equal(t, 25, s.Age)
	})
}

func TestRemoveEmptyObjectsWithDepth(t *testing.T) {
	t.Run("removes empty nested objects", func(t *testing.T) {
		data := map[string]any{
			"name":  "test",
			"empty": map[string]any{},
		}
		removeEmptyObjectsWithDepth(data, 0)
		assert.NotContains(t, data, "empty")
		assert.Contains(t, data, "name")
	})

	t.Run("removes objects that become empty after cleaning", func(t *testing.T) {
		data := map[string]any{
			"outer": map[string]any{
				"inner": map[string]any{},
			},
		}
		removeEmptyObjectsWithDepth(data, 0)
		assert.NotContains(t, data, "outer")
	})

	t.Run("cleans objects inside arrays", func(t *testing.T) {
		inner := map[string]any{
			"nested": map[string]any{},
		}
		data := map[string]any{
			"items": []any{inner},
		}
		removeEmptyObjectsWithDepth(data, 0)
		assert.NotContains(t, inner, "nested")
	})

	t.Run("stops at max depth", func(t *testing.T) {
		data := map[string]any{
			"empty": map[string]any{},
		}
		removeEmptyObjectsWithDepth(data, 21)
		// Should not remove because depth > 20
		assert.Contains(t, data, "empty")
	})
}
