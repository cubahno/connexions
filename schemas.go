package xs

import (
	"github.com/getkin/kin-openapi/openapi3"
	"strings"
)

func GetContentType(content openapi3.Content) (string, *openapi3.Schema) {
	prioTypes := []string{"application/json", "text/plain", "text/html"}
	for _, contentType := range prioTypes {
		if _, ok := content[contentType]; ok {
			return contentType, content[contentType].Schema.Value
		}
	}

	for contentType, mediaType := range content {
		return contentType, mediaType.Schema.Value
	}

	return "", nil
}

func CollectParams(params openapi3.Parameters) map[string]map[string]*openapi3.Parameter {
	res := make(map[string]map[string]*openapi3.Parameter)

	for _, paramRef := range params {
		param := paramRef.Value
		if param == nil {
			continue
		}

		in := strings.ToLower(param.In)
		if len(res[in]) == 0 {
			res[in] = make(map[string]*openapi3.Parameter)
		}
		res[in][param.Name] = param
	}
	return res
}

func nestedSchema(schema *openapi3.Schema) interface{} {
	if schema.Properties == nil {
		return []interface{}{}
	}

	d := make(map[string]interface{})
	for key, prop := range schema.Properties {
		d[key] = nestedSchema(prop.Value)
	}
	if schema.AllOf != nil {
		for _, schemaRef := range schema.AllOf {
			schema2 := schemaRef.Value
			for key, prop := range schema2.Properties {
				if _, ok := d[key]; !ok {
					d[key] = nestedSchema(prop.Value)
				}
			}
		}
	}
	return d
}
