package xs

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"testing"
)

func CreateSchemaFromString(t *testing.T, src string) *openapi3.Schema {
	schema := &openapi3.Schema{}
	err := json.Unmarshal([]byte(src), schema)
	if err != nil {
		t.FailNow()
	}
	return schema
}
