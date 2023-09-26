package connexions

import "errors"

var (
	ErrServiceNotFound                  = errors.New("service not found")
	ErrResourceNotFound                 = errors.New("resource not found")
	ErrResourceMethodNotFound           = errors.New("resource method not found")
	ErrOpenAPISpecIsEmpty               = errors.New("OpenAPI spec is empty")
	ErrInvalidHTTPVerb                  = errors.New("invalid HTTP verb")
	ErrInvalidURLResource               = errors.New("invalid URL resource")
	ErrReservedPrefix                   = errors.New("reserved prefix")
	ErrUnexpectedFormDataType           = errors.New("expected map[string]any for multipart/form-data")
	ErrUnexpectedFormURLEncodedType     = errors.New("expected map[string]any for x-www-form-urlencoded")
	ErrOnlyFixedResourcesAllowedEditing = errors.New("only fixed resources are allowed editing")
	ErrGettingFileFromURL               = errors.New("error getting file from url")
)
