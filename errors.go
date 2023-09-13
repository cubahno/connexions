package connexions

import "errors"

var (
	ErrInvalidConfig                = errors.New("invalid config")
	ErrServiceNotFound              = errors.New("service not found")
	ErrResourceNotFound             = errors.New("resource not found")
	ErrResourceMethodNotFound       = errors.New("resource method not found")
	ErrOpenAPISpecIsEmpty           = errors.New("OpenAPI spec is empty")
	ErrInvalidHTTPVerb              = errors.New("invalid HTTP verb")
	ErrInvalidURLResource           = errors.New("invalid URL resource")
	ErrCreatingDirectories          = errors.New("error creating directories")
	ErrCreatingFile                 = errors.New("error creating file")
	ErrReservedPrefix               = errors.New("reserved prefix")
	ErrNoPathsInSchema              = errors.New("no paths found in schema")
	ErrUnexpectedFormDataType       = errors.New("expected map[string]any for multipart/form-data")
	ErrUnexpectedFormURLEncodedType = errors.New("expected map[string]any for x-www-form-urlencoded")
)
