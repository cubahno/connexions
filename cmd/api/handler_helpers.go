package api

import (
	"fmt"
	"strings"

	"github.com/cubahno/connexions/v2/pkg/schema"
	"github.com/doordash-oss/oapi-codegen-dd/v3/pkg/codegen"
)

const schemaInit = "schema.Schema"

// renderParameterEncoding renders a ParameterEncoding struct as Go code
func renderParameterEncoding(enc *codegen.ParameterEncoding) string {
	if enc == nil || isEmptyEncoding(enc) {
		return "nil"
	}

	var parts []string

	if enc.Style != "" {
		parts = append(parts, fmt.Sprintf("Style: %q", enc.Style))
	}

	if enc.Explode != nil {
		explodeVal := "false"
		if *enc.Explode {
			explodeVal = "true"
		}
		parts = append(parts, fmt.Sprintf("Explode: &[]bool{%s}[0]", explodeVal))
	}

	if enc.Required {
		parts = append(parts, "Required: true")
	}

	if enc.AllowReserved {
		parts = append(parts, "AllowReserved: true")
	}

	return fmt.Sprintf("&codegen.ParameterEncoding{%s}", strings.Join(parts, ", "))
}

// isEmptyEncoding checks if a ParameterEncoding has any non-default values
func isEmptyEncoding(enc *codegen.ParameterEncoding) bool {
	return enc.Style == "" && enc.Explode == nil && !enc.Required && !enc.AllowReserved
}

// renderSchema renders a schema.Schema as Go code
func renderSchema(s *schema.Schema) string {
	if s == nil {
		return "nil"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("&%s{\n", schemaInit))

	if s.Type != "" {
		sb.WriteString(fmt.Sprintf("\t\tType: %q,\n", s.Type))
	}

	if s.Format != "" {
		sb.WriteString(fmt.Sprintf("\t\tFormat: %q,\n", s.Format))
	}

	if s.Pattern != "" {
		sb.WriteString(fmt.Sprintf("\t\tPattern: %q,\n", s.Pattern))
	}

	if s.Minimum != nil {
		sb.WriteString(fmt.Sprintf("\t\tMinimum: ptr(%v),\n", *s.Minimum))
	}

	if s.Maximum != nil {
		sb.WriteString(fmt.Sprintf("\t\tMaximum: ptr(%v),\n", *s.Maximum))
	}

	if s.MinLength != nil {
		sb.WriteString(fmt.Sprintf("\t\tMinLength: ptr(int64(%v)),\n", *s.MinLength))
	}

	if s.MaxLength != nil {
		sb.WriteString(fmt.Sprintf("\t\tMaxLength: ptr(int64(%v)),\n", *s.MaxLength))
	}

	if s.MinItems != nil {
		sb.WriteString(fmt.Sprintf("\t\tMinItems: ptr(int64(%v)),\n", *s.MinItems))
	}

	if s.MaxItems != nil {
		sb.WriteString(fmt.Sprintf("\t\tMaxItems: ptr(int64(%v)),\n", *s.MaxItems))
	}

	if len(s.Enum) > 0 {
		sb.WriteString("\t\tEnum: []any{")
		for i, e := range s.Enum {
			if i > 0 {
				sb.WriteString(", ")
			}
			// Handle different types of enum values
			switch v := e.(type) {
			case string:
				sb.WriteString(fmt.Sprintf("%q", v))
			case bool, int, int64, float64:
				sb.WriteString(fmt.Sprintf("%v", v))
			default:
				sb.WriteString(fmt.Sprintf("%q", fmt.Sprint(v)))
			}
		}
		sb.WriteString("},\n")
	}

	if s.Items != nil {
		sb.WriteString(fmt.Sprintf("\t\tItems: %s,\n", renderSchema(s.Items)))
	}

	if len(s.Properties) > 0 {
		sb.WriteString(fmt.Sprintf("\t\tProperties: map[string]*%s{\n", schemaInit))
		for name, prop := range s.Properties {
			sb.WriteString(fmt.Sprintf("\t\t\t%q: %s,\n", name, renderSchema(prop)))
		}
		sb.WriteString("\t\t},\n")
	}

	if len(s.Required) > 0 {
		sb.WriteString("\t\tRequired: []string{")
		for i, r := range s.Required {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%q", r))
		}
		sb.WriteString("},\n")
	}

	if s.AdditionalProperties != nil {
		sb.WriteString(fmt.Sprintf("\t\tAdditionalProperties: %s,\n", renderSchema(s.AdditionalProperties)))
	}

	if s.Nullable {
		sb.WriteString("\t\tNullable: true,\n")
	}

	if s.ReadOnly {
		sb.WriteString("\t\tReadOnly: true,\n")
	}

	if s.WriteOnly {
		sb.WriteString("\t\tWriteOnly: true,\n")
	}

	if s.Deprecated {
		sb.WriteString("\t\tDeprecated: true,\n")
	}

	sb.WriteString("\t\t}")
	return sb.String()
}

// createResolveTypeName returns a function that resolves a type name to its final non-ref type
// by following the RefType chain until it finds a concrete type.
func createResolveTypeName(tdsLookUp map[string]*codegen.TypeDefinition) func(typeName string) string {
	return func(typeName string) string {
		visited := make(map[string]bool)
		current := typeName

		// Follow the RefType chain until we find a non-ref type
		for {
			// Prevent infinite loops
			if visited[current] {
				return current
			}
			visited[current] = true

			// Look up the type definition
			td, ok := tdsLookUp[current]
			if !ok {
				// Type not found in registry, return current name
				return current
			}

			// If this type has a RefType, follow it
			if td.Schema.RefType != "" {
				current = td.Schema.RefType
				continue
			}

			// No RefType, this is the final concrete type
			return current
		}
	}
}
