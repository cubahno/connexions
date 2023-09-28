package main

import (
	"errors"
	"flag"
	"github.com/cubahno/connexions"
	"github.com/getkin/kin-openapi/openapi3"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	ErrNotRegularFile = errors.New("not a regular file")
)

func main() {
	var src string
	var dst string

	flag.StringVar(&src, "src", "", "path to source openapi file")
	flag.StringVar(&dst, "dst", "", "path to destination directory or file")
	flag.Parse()

	if len(flag.Args()) == 2 && src == "" && dst == "" {
		src = flag.Args()[0]
		dst = flag.Args()[1]
	}

	if src == "" || dst == "" {
		log.Println("src and dst flags are required")
		return
	}

	err := processFile(src, dst)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("src: %s, dst: %s\n", src, dst)
}

func processFile(src, dest string) error {
	t1 := time.Now()

	doc, err := getSourceDocument(src)
	if err != nil {
		return err
	}

	type parsed struct {
		resource string
		method   string
		item     *openapi3.Operation
	}

	ch := make(chan parsed)
	var wg sync.WaitGroup
	num := 0

	for resName, resMethods := range doc.GetResources() {
		num += 1
		for _, method := range resMethods {
			wg.Add(1)

			num += 1
			go func(resName, method string) {
				defer wg.Done()

				log.Printf("processing %s %s\n", method, resName)

				operation := doc.FindOperation(&connexions.OperationDescription{
					Resource: resName,
					Method:   method,
				})
				operation = operation.WithParseConfig(&connexions.ParseConfig{
					MaxRecursionLevels: 0,
					OnlyRequired:       true,
				})
				item := convertOperation(operation)

				ch <- parsed{
					resource: resName,
					method:   method,
					item:     item,
				}
			}(resName, method)
		}
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	log.Printf("waiting for %d items\n", num)

	d := &openapi3.T{
		OpenAPI: doc.GetVersion(),
		Info:    &openapi3.Info{},
	}

	for itemRes := range ch {
		d.AddOperation(itemRes.resource, itemRes.method, itemRes.item)
	}

	contents, err := d.MarshalJSON()
	if err != nil {
		return err
	}

	err = connexions.SaveFile(dest, contents)
	if err == nil {
		t2 := time.Now()
		log.Printf("[%s]: done in %v\n", src, t2.Sub(t1))
	}

	return err
}

func getSourceDocument(src string) (connexions.Document, error) {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil, err
	}

	log.Printf("processing file: %s\n", src)

	fileInfo, err := os.Stat(src)
	if err != nil {
		return nil, err
	}
	if !fileInfo.Mode().IsRegular() {
		return nil, ErrNotRegularFile
	}

	return connexions.NewDocumentFromFileFactory(connexions.LibOpenAPIProvider)(src)
}

func convertOperation(operation connexions.Operationer) *openapi3.Operation {
	reqBody, reqContentType := operation.GetRequestBody()
	response := operation.GetResponse()

	var requestBody *openapi3.RequestBodyRef
	if reqBody != nil && reqContentType != "" {
		requestBody = &openapi3.RequestBodyRef{
			Value: &openapi3.RequestBody{
				Content: map[string]*openapi3.MediaType{
					reqContentType: {
						Schema: convertSchema(reqBody),
					},
				},
			},
		}
	}

	contentSchema := response.Content
	contentType := response.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	var params openapi3.Parameters
	for _, param := range operation.GetParameters() {
		params = append(params, &openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name:     param.Name,
				In:       param.In,
				Required: param.Required,
				Example:  param.Example,
				Schema:   convertSchema(param.Schema),
			},
		})
	}

	return &openapi3.Operation{
		OperationID: operation.ID(),
		Parameters:  params,
		RequestBody: requestBody,
		Responses: openapi3.Responses{
			strconv.Itoa(response.StatusCode): {
				Value: &openapi3.Response{
					Content: map[string]*openapi3.MediaType{
						contentType: {
							Schema: convertSchema(contentSchema),
						},
					},
				},
			},
		},
	}
}

func convertSchema(src *connexions.Schema) *openapi3.SchemaRef {
	if src == nil {
		return nil
	}

	var props openapi3.Schemas
	if src.Properties != nil {
		props = make(openapi3.Schemas)
		for k, v := range src.Properties {
			props[k] = convertSchema(v)
		}
	}

	asUint64Ptr := func(v int64) *uint64 {
		if v == 0 {
			return nil
		}
		u := uint64(v)
		return &u
	}

	asFloat64Ptr := func(v float64) *float64 {
		if v == 0 {
			return nil
		}
		return &v
	}

	return &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Not:        convertSchema(src.Not),
			Type:       src.Type,
			Format:     src.Format,
			Enum:       src.Enum,
			Default:    src.Default,
			Example:    src.Example,
			Nullable:   src.Nullable,
			ReadOnly:   src.ReadOnly,
			WriteOnly:  src.WriteOnly,
			Deprecated: src.Deprecated,
			Min:        asFloat64Ptr(src.Minimum),
			Max:        asFloat64Ptr(src.Maximum),
			MultipleOf: asFloat64Ptr(src.MultipleOf),
			MinLength:  uint64(src.MinLength),
			MaxLength:  asUint64Ptr(src.MaxLength),
			Pattern:    src.Pattern,
			MinItems:   uint64(src.MinItems),
			MaxItems:   asUint64Ptr(src.MaxItems),
			Items:      convertSchema(src.Items),
			Required:   src.Required,
			Properties: props,
			MinProps:   uint64(src.MinProperties),
			MaxProps:   asUint64Ptr(src.MaxProperties),
		},
	}
}
