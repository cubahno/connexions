package connexions

import "errors"

var (
	ErrInvalidConfig                    = errors.New("invalid config")
	ErrServiceNotFound                  = errors.New("service not found")
	ErrResourceNotFound                 = errors.New("resource not found")
	ErrResourceMethodNotFound           = errors.New("resource method not found")
	ErrOpenAPISpecIsEmpty               = errors.New("OpenAPI spec is empty")
	ErrInvalidHTTPVerb                  = errors.New("invalid HTTP verb")
	ErrInvalidURLResource               = errors.New("invalid URL resource")
	ErrCreatingDirectories              = errors.New("error creating directories")
	ErrCreatingFile                     = errors.New("error creating file")
	ErrReservedPrefix                   = errors.New("reserved prefix")
	ErrNoPathsInSchema                  = errors.New("no paths found in schema")
	ErrUnexpectedFormDataType           = errors.New("expected map[string]any for multipart/form-data")
	ErrUnexpectedFormURLEncodedType     = errors.New("expected map[string]any for x-www-form-urlencoded")
	ErrOnlyFixedResourcesAllowedEditing = errors.New("only fixed resources are allowed editing")
	ErrInternalServer                   = errors.New("internal server error")
	ErrFileUpload                       = errors.New("error uploading file")
	ErrExtractingFiles                  = errors.New("error extracting files")
	ErrReadingZipFile                   = errors.New("error reading zip file")
)
