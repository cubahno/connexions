package types

import (
	"github.com/doordash-oss/oapi-codegen-dd/v3/pkg/codegen"
	"github.com/doordash-oss/oapi-codegen-dd/v3/pkg/runtime"
)

// EncodeFormData encodes data as application/x-www-form-urlencoded using the provided encoding metadata.
// It converts codegen.RequestBodyEncoding to runtime.FieldEncoding and calls the runtime encoder.
func EncodeFormData(data any, encoding map[string]codegen.RequestBodyEncoding) (string, error) {
	if encoding == nil {
		encoding = make(map[string]codegen.RequestBodyEncoding)
	}

	// Convert codegen.RequestBodyEncoding to runtime.FieldEncoding
	runtimeEncoding := make(map[string]runtime.FieldEncoding, len(encoding))
	for key, enc := range encoding {
		runtimeEncoding[key] = runtime.FieldEncoding{
			Style:       enc.Style,
			Explode:     enc.Explode,
			ContentType: enc.ContentType,
		}
	}

	return runtime.EncodeFormFields(data, runtimeEncoding)
}
