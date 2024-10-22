package lib

import (
	"context"
	"encoding/json"
	base2 "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func AssertJSONEqual(t *testing.T, expected, actual any) {
	t.Helper()
	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	// Compare JSON representations
	assert.Equal(t, string(expectedJSON), string(actualJSON), "JSON representations should match")
}

func CreateLibSchemaFromString(t *testing.T, ymlSchema string) *base2.SchemaProxy {
	t.Helper()
	// unmarshal raw bytes
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(ymlSchema), &node)

	// build out the low-level model
	var lowSchema base.SchemaProxy
	_ = low.BuildModel(node.Content[0], &lowSchema)
	_ = lowSchema.Build(context.TODO(), node.Content[0], nil, nil)

	// build the high level schema proxy
	return base2.NewSchemaProxy(&low.NodeReference[*base.SchemaProxy]{
		Value: &lowSchema,
	})
}
