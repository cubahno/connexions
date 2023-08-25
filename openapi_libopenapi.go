package connexions

import (
    "fmt"
    "github.com/pb33f/libopenapi"
    "github.com/pb33f/libopenapi/datamodel/high/base"
    v2high "github.com/pb33f/libopenapi/datamodel/high/v2"
    v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
    "net/http"
    "os"
    "strings"
)

type LibV2Document struct {
    libopenapi.DocumentModel[v2high.Swagger]
}

type LibV3Document struct {
    libopenapi.DocumentModel[v3high.Document]
}

type LibV3Operation struct {
    *v3high.Operation
}

type LibV3Response struct {
    *v3high.Response
}

func (d *LibV3Document) GetVersion() string {
    return d.GetVersion()
}

func (d *LibV3Document) GetResources() map[string][]string {
    res := make(map[string][]string)

    for name, path := range d.Model.Paths.PathItems {
        res[name] = make([]string, 0)
        for method, _ := range path.GetOperations() {
            res[name] = append(res[name], strings.ToUpper(method))
        }
    }
    return res
}

func (d *LibV3Document) FindOperation(resourceName, method string) Operationer {
    path, ok  := d.Model.Paths.PathItems[resourceName]
    if !ok {
        return nil
    }

    for m, op := range path.GetOperations() {
        if strings.ToUpper(m) == strings.ToUpper(method) {
            return &LibV3Operation{op}
        }
    }

    return nil
}

func (op *LibV3Operation) GetParameters() OpenAPIParameters {
    params := make(OpenAPIParameters, 0)

    for _, param := range op.Parameters {
        var schema *Schema
        if param.Schema != nil {
            schema = NewSchemaFromLibOpenAPI(param.Schema.Schema(), nil)
        }

        params = append(params, &OpenAPIParameter{
            Name: param.Name,
            In: param.In,
            Required: param.Required,
            Schema: schema,
            Example: param.Example,
        })
    }

    return params
}

func (op *LibV3Operation) GetResponse() (OpenAPIResponse, int) {
    available := op.Responses.Codes

    var responseRef *v3high.Response
    for _, code := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
        responseRef = available[fmt.Sprintf("%v", code)]
        if responseRef != nil {
            return &LibV3Response{responseRef}, code
        }
    }

    // Get first defined
    for codeName, respRef := range available {
        if codeName == "default" {
            continue
        }
        return &LibV3Response{respRef}, TransformHTTPCode(codeName)
    }

    return &LibV3Response{op.Responses.Default}, 200
}

func (op *LibV3Operation) GetRequestBody() (*Schema, string) {
    if op.RequestBody == nil {
        return nil, ""
    }

    contentTypes := op.RequestBody.Content
    if len(contentTypes) == 0 {
        return nil, ""
    }

    typesOrder := []string{"application/json", "multipart/form-data", "application/x-www-form-urlencoded"}
    for _, contentType := range typesOrder {
        if _, ok := contentTypes[contentType]; ok {
            return NewSchemaFromLibOpenAPI(contentTypes[contentType].Schema.Schema(), nil), contentType
        }
    }

    // Get first defined
    for contentType, mediaType := range contentTypes {
        return NewSchemaFromLibOpenAPI(mediaType.Schema.Schema(), nil), contentType
    }

    return nil, ""
}

func (r *LibV3Response) GetContent() (string, *Schema) {
    types := r.Content
    if len(types) == 0 {
        return "", nil
    }

    prioTypes := []string{"application/json", "text/plain", "text/html"}
    for _, contentType := range prioTypes {
        if _, ok := types[contentType]; ok {
            return contentType, NewSchemaFromLibOpenAPI(types[contentType].Schema.Schema(), nil)
        }
    }

    for contentType, mediaType := range types {
        return contentType, NewSchemaFromLibOpenAPI(mediaType.Schema.Schema(), nil)
    }

    return "", nil
}

func (r *LibV3Response) GetHeaders() OpenAPIHeaders {
    res := make(OpenAPIHeaders)
    for name, header := range r.Headers {
        if header == nil {
            continue
        }

        var schema *Schema
        if header.Schema != nil {
            schema = NewSchemaFromLibOpenAPI(header.Schema.Schema(), nil)
        }

        res[name] = &OpenAPIParameter{
            Name:     name,
            In:       ParameterInHeader,
            Required: header.Required,
            Schema:   schema,
        }
    }
    return res
}

func NewSchemaFromLibOpenAPI(s *base.Schema, visited map[string]bool) *Schema {
    if s == nil {
        return nil
    }

    // s = MergeKinSubSchemas(s)

    if len(visited) == 0 {
        visited = make(map[string]bool)
    }

    var items *SchemaWithReference
    if s.Items != nil && s.Items.IsA() {
        libItems := s.Items.A
        ref := libItems.GetReference()
        if ref != "" {
            if visited[ref] {
                return nil
            }

            visited[ref] = true
        }
        items = &SchemaWithReference{
            Schema:    NewSchemaFromLibOpenAPI(libItems.Schema(), visited),
            Reference: ref,
        }
    }

    var properties map[string]*SchemaWithReference
    if len(s.Properties) > 0 {
        properties = make(map[string]*SchemaWithReference)
        for name, sProxy := range s.Properties {
            t := visited
            ref := sProxy.GetReference()
            if ref != "" && visited[ref] {
                continue
            }

            if ref != "" {
                visited[ref] = true
            }

            properties[name] = &SchemaWithReference{
                Schema:    NewSchemaFromLibOpenAPI(sProxy.Schema(), t),
                Reference: ref,
            }
        }
    }

    typ := ""
    if len(s.Type) >0 {
        typ = s.Type[0]
    }

    return &Schema{
        Type:          typ,
        Items:         items,
        MultipleOf:    RemovePointer(s.MultipleOf),
        Maximum:       RemovePointer(s.Maximum),
        Minimum:       RemovePointer(s.Minimum),
        MaxLength:     RemovePointer(s.MaxLength),
        MinLength:     RemovePointer(s.MinLength),
        Pattern:       s.Pattern,
        Format:        s.Format,
        MaxItems:      RemovePointer(s.MaxItems),
        MinItems:      RemovePointer(s.MinItems),
        MaxProperties: RemovePointer(s.MaxProperties),
        MinProperties: RemovePointer(s.MinProperties),
        Required:      s.Required,
        Enum:          s.Enum,
        Properties:    properties,
        Default:       s.Default,
        Nullable:      RemovePointer(s.Nullable),
        ReadOnly:      s.ReadOnly,
        WriteOnly:     s.WriteOnly,
        Example:       s.Example,
        Deprecated:    RemovePointer(s.Deprecated),
    }
}

func NewLibDocument(filePath string) (libopenapi.Document, error) {
    src, _ := os.ReadFile(filePath)

    // create a new document from specification bytes
    return libopenapi.NewDocument(src)
}

func NewLibModel(doc libopenapi.Document) (*libopenapi.DocumentModel[v3high.Document], []error) {
    return doc.BuildV3Model()
}
