package xs

import "testing"

func TestReplaceMapFunctionPlaceholders(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"id": "func:makeUUID",
			"country": map[string]interface{}{
				"id":   "func:makeUUID",
				"code": "func:makeCountryCode",
			},
			"name":  "Jane Doe",
			"email": "",
		},
	}

	funcs := map[string]interface{}{
		"makeUUID":        func() any { return "GeneratedUUID" },
		"makeCountryCode": func() any { return "DE" },
	}

	result := ReplaceMapFunctionPlaceholders(data, funcs)

	if user, ok := result.(map[string]any)["user"].(map[string]any); ok {
		if idFn, ok := user["id"].(func() any); ok {
			idValue := idFn()
			if idValue != "GeneratedUUID" {
				t.Errorf("Expected GeneratedUUID, got %s", idValue)
			}
		} else {
			t.Error("Failed to get function for user id")
		}

		if countryCodeFn, ok := user["country"].(map[string]any)["code"].(func() any); ok {
			codeValue := countryCodeFn()
			if codeValue != "DE" {
				t.Errorf("Expected DE, got %s", codeValue)
			}
		} else {
			t.Error("Failed to get function for country code")
		}
	} else {
		t.Error("Failed to get user map")
	}
}
