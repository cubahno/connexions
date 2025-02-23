package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/files"
	"github.com/cubahno/connexions/internal/openapi"
	"github.com/cubahno/connexions/internal/types"
	"github.com/getkin/kin-openapi/openapi3"
)

var (
	ErrNotRegularFile = errors.New("not a regular file")
)

type parseConfig struct {
	maxRecursionLevels int
	onlyRequired       bool
	replace            bool
	take               map[string]bool
}

// Custom type for a string slice flag
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, " ")
}

func (s *stringSlice) Set(value string) error {
	value = strings.Trim(value, " ")
	if value == "" {
		return nil
	}
	*s = append(*s, strings.Split(value, " ")...)
	return nil
}

func main() {
	var src string
	var dst string
	var rpl string
	var onlyRequired bool
	var maxRecursion int
	var takes stringSlice

	flag.StringVar(&src, "src", "", "path to source openapi file")
	flag.StringVar(&dst, "dst", "", "path to destination directory or file")
	flag.StringVar(&rpl, "replace", "", "replace source file with simplified version. new version will have .simpl.json extension")
	flag.BoolVar(&onlyRequired, "only-required", false, "discard non-required fields")
	flag.IntVar(&maxRecursion, "max-recursion", 0, "maximum recursion levels (0 - unlimited)")
	flag.Var(&takes, "take", "Space separated values or multiple --take flags")

	flag.Parse()

	if len(flag.Args()) == 2 && src == "" && dst == "" {
		src = flag.Args()[0]
		dst = flag.Args()[1]
	}

	replace := false
	if strings.ToLower(rpl) == "true" {
		replace = true
	}

	takeMap := make(map[string]bool)
	for _, v := range takes {
		v = strings.Trim(v, " \n")
		takeMap[v] = true
	}

	cfg := &parseConfig{
		maxRecursionLevels: maxRecursion,
		onlyRequired:       onlyRequired,
		replace:            replace,
		take:               takeMap,
	}

	if src == "" {
		log.Println("src flag is required")
		return
	}

	// process single file
	fileInfo, _ := os.Stat(src)
	if fileInfo != nil && !fileInfo.IsDir() {
		dst = getDestPath(filepath.Base(src), filepath.Base(src), dst)
		err := processFile(src, dst, cfg)
		if err != nil {
			log.Println(err)
		}
		return
	}

	sources, err := collectSources(src)
	if err != nil {
		log.Println(err)
		return
	}

	if err := run(src, sources, dst, cfg); err != nil {
		log.Println(err)
		return
	}
}

// collectSources collects relative paths
func collectSources(src string) ([]string, error) {
	fileInfo, err := os.Stat(src)
	if os.IsNotExist(err) {
		return nil, err
	}

	// single file passed
	if !fileInfo.IsDir() {
		return []string{src}, nil
	}

	// we have a directory to walk
	var files []string

	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// skip done files
		if strings.HasSuffix(info.Name(), ".simpl.json") {
			return nil
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(src, path)
			files = append(files, rel)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func run(baseSrcPath string, sources []string, dst string, cfg *parseConfig) error {
	type result struct {
		src string
		err error
	}

	var wg sync.WaitGroup
	ch := make(chan result)

	for _, src := range sources {
		wg.Add(1)

		go func(src, dst string) {
			defer wg.Done()

			dstPath := getDestPath(baseSrcPath, src, dst)
			err := processFile(filepath.Join(baseSrcPath, src), dstPath, cfg)
			ch <- result{
				src: src,
				err: err,
			}
		}(src, dst)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for res := range ch {
		msg := fmt.Sprintf("[%s]: ", res.src)
		if res.err != nil {
			msg += res.err.Error()
		} else {
			msg += "done"
		}
		log.Println(msg)
	}

	return nil
}

func getDestPath(baseSrcPath, relFilePath, dst string) string {
	if dst == "" {
		dst = baseSrcPath
	}
	dst = filepath.Join(dst, relFilePath)
	currentExt := filepath.Ext(dst)
	return filepath.Join(filepath.Dir(dst), fmt.Sprintf("%s.simpl.json", filepath.Base(dst[:len(dst)-len(currentExt)])))
}

func processFile(src, dest string, cfg *parseConfig) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[%s]: %v\n", src, r)
		}
	}()

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

	securityComponents := doc.GetSecurity()

	for resName, resMethods := range doc.GetResources() {
		if len(cfg.take) > 0 && !cfg.take[resName] {
			continue
		}
		num += 1
		for _, method := range resMethods {
			wg.Add(1)

			num += 1
			go func(resName, method string) {
				defer wg.Done()

				operation := doc.FindOperation(&openapi.OperationDescription{
					Resource: resName,
					Method:   method,
				})
				operation = operation.WithParseConfig(&config.ParseConfig{
					MaxRecursionLevels: cfg.maxRecursionLevels,
					OnlyRequired:       cfg.onlyRequired,
				})
				item := convertOperation(operation, securityComponents)

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

	err = files.SaveFile(dest, contents)
	if cfg.replace {
		_ = os.Remove(src)
	}

	return err
}

func getSourceDocument(src string) (openapi.Document, error) {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil, err
	}

	fileInfo, err := os.Stat(src)
	if err != nil {
		return nil, err
	}
	if !fileInfo.Mode().IsRegular() {
		return nil, ErrNotRegularFile
	}

	return openapi.NewDocumentFromFile(src)
}

func convertOperation(operation openapi.Operation, securityComponents openapi.SecurityComponents) *openapi3.Operation {
	request := operation.GetRequest(securityComponents)
	payload := request.Body
	reqBody := payload.Schema
	reqContentType := payload.Type

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
	for _, param := range request.Parameters {
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
		Responses: openapi3.NewResponses(openapi3.WithStatus(response.StatusCode, &openapi3.ResponseRef{
			Value: &openapi3.Response{
				Content: map[string]*openapi3.MediaType{
					contentType: {
						Schema: convertSchema(contentSchema),
					},
				},
			},
		})),
	}
}

func convertSchema(src *types.Schema) *openapi3.SchemaRef {
	if src == nil {
		return nil
	}

	var props openapi3.Schemas
	if src.Properties != nil {
		props = make(openapi3.Schemas)
		for k, v := range src.Properties {
			res := convertSchema(v)
			if res != nil {
				props[k] = res
			}
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
			Type:       &openapi3.Types{src.Type},
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
