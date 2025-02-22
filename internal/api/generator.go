package api

import (
	"net/http"
	"strings"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/openapi"
	"github.com/cubahno/connexions/internal/replacer"
)

type generateResourceOptions struct {
	config        *config.ServiceConfig
	valueReplacer replacer.ValueReplacer
	withRequest   bool
	withResponse  bool
	req           *http.Request
}

func generateResource(service *ServiceItem, rd *RouteDescription, opts *generateResourceOptions) (*GenerateResponse, error) {
	fileProps := rd.File
	if fileProps == nil {
		return nil, ErrResourceNotFound
	}

	var genReq *openapi.GeneratedRequest
	var genResp *openapi.GeneratedResponse

	if !fileProps.IsOpenAPI {
		if opts.withRequest {
			genReq = openapi.NewRequestFromFixedResource(
				fileProps.Prefix+fileProps.Resource,
				fileProps.Method,
				fileProps.ContentType,
				opts.valueReplacer,
			)
		}
		if opts.withResponse {
			genResp = openapi.NewResponseFromFixedResource(
				fileProps.FilePath, fileProps.ContentType, opts.valueReplacer)
		}

		return &GenerateResponse{
			Request:  genReq,
			Response: genResp,
		}, nil
	}

	spec := fileProps.Spec
	operation := spec.FindOperation(&openapi.OperationDescription{
		Service:  service.Name,
		Resource: rd.Path,
		Method:   strings.ToUpper(rd.Method),
	})

	if operation == nil {
		return nil, ErrResourceMethodNotFound
	}
	operation = operation.WithParseConfig(opts.config.ParseConfig)

	requestOptions := &openapi.GenerateRequestOptions{
		PathPrefix: fileProps.Prefix,
		Path:       rd.Path,
		Method:     rd.Method,
		Operation:  operation,
	}

	if opts.withRequest {
		genReq = openapi.NewRequestFromOperation(requestOptions, spec.GetSecurity(), opts.valueReplacer)
	}

	if opts.withResponse {
		genResp = openapi.NewResponseFromOperation(operation, opts.valueReplacer, opts.req)
	}

	return &GenerateResponse{
		Request:  genReq,
		Response: genResp,
	}, nil
}
