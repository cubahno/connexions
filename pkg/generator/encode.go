package generator

import (
	"encoding/json"
	"encoding/xml"
	"fmt"

	"go.yaml.in/yaml/v4"
)

func encodeContent(content any, contentType string) ([]byte, error) {
	if content == nil {
		return nil, nil
	}

	switch contentType {
	case "application/json", "":
		// Empty content-type defaults to JSON
		return json.Marshal(content)

	case "application/x-www-form-urlencoded",
		"multipart/form-data",
		"multipart/formdata":
		// For mock server: return JSON for easy debugging/display in browser dev tools.
		// Real servers would return proper URL-encoded (key1=value1&key2=value2) or
		// multipart format with boundaries, but JSON is more practical for development.
		res, err := json.Marshal(content)
		if err != nil {
			return nil, err
		}
		if string(res) == "{}" {
			res = []byte("")
		}
		return res, nil

	case "application/xml":
		return xml.Marshal(content)

	case "application/x-yaml":
		return yaml.Dump(content, yaml.WithIndent(2))

	default:
		switch v := content.(type) {
		case []byte:
			return v, nil
		case string:
			return []byte(v), nil
		}
	}

	return nil, fmt.Errorf("cannot encode type %T with content-type %s", content, contentType)
}
