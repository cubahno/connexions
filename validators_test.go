package connexions

import "testing"

func TestIsValidURLResource(t *testing.T) {
	tests := []struct {
		url    string
		expect bool
	}{
		{"", true},
		{"/", true},
		{"abc", true},
		{"{user-id}", true},
		{"{file_id}", true},
		{"{some_name_1}", true},
		{"{id}/{name}", true},
		{"/users/{id}/files/{file_id}", true},
		{"{!@#$%^}", false},
		{"{name:str}/{id}", false},
		{"{name?str}/{id}", false},
		{"{}", false},
		{"{}/{}", false},
	}

	for _, test := range tests {
		result := IsValidURLResource(test.url)
		if result != test.expect {
			t.Errorf("For URL Pattern: %s, Expected: %v, Got: %v", test.url, test.expect, result)
		}
	}
}
