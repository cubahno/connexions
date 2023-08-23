package connexions

import "testing"

func TestGetValueByDottedPath(t *testing.T) {
	data := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": map[string]interface{}{
				"var": "hello",
			},
		},
	}

	tests := []struct {
		dottedPath string
		expected   interface{}
	}{
		{"foo.bar.var", "hello"},
		{"foo.bar.nonexistent", nil},
		{"foo.nonexistent.var", nil},
		{"nonexistent.bar.var", nil},
	}

	for _, test := range tests {
		result := GetValueByDottedPath(data, test.dottedPath)
		if result != test.expected {
			t.Errorf("For path %s, expected %v but got %v", test.dottedPath, test.expected, result)
		}
	}
}

func TestSetValueByDottedPath(t *testing.T) {
	tests := []struct {
		dottedPath string
		value      any
		src        map[string]any
		expected   any
	}{
		{"nice.dice.quite", "rice",
			map[string]any{
				"nice": map[string]any{
					"dice": map[string]any{
						"other": "mice",
						"quite": "not rice"}}},
			map[string]any{
				"nice": map[string]any{
					"dice": map[string]any{
						"other": "mice",
						"quite": "rice"}}},
		},
		{"foo.bar.var", "new value",
			map[string]any{},
			map[string]any{
				"foo": map[string]any{
					"bar": map[string]any{
						"var": "new value"}}}},
	}

	for _, test := range tests {
		data := test.src
		SetValueByDottedPath(data, test.dottedPath, test.value)

		result := GetValueByDottedPath(data, test.dottedPath)

		if result != test.value {
			t.Errorf("For path %s, expected %v but got %v", test.dottedPath, test.value, result)
		}
	}
}

func TestGetRandomKeyFromMap(t *testing.T) {
	myMap := map[string]bool{
		"key1": true,
		"key2": true,
		"key3": true,
	}

	randomKey := GetRandomKeyFromMap(myMap)

	if randomKey == "" {
		t.Errorf("Expected a non-empty random key, but got an empty key")
	}

	if _, exists := myMap[randomKey]; !exists {
		t.Errorf("Random key %s does not exist in the original map", randomKey)
	}
}
