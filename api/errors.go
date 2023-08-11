package api

import "errors"

var (
	ErrResourceNotFound       = errors.New("resource not found")
	ErrResourceMethodNotFound = errors.New("resource method not found")
	ErrOpenAPISpecIsEmpty     = errors.New("OpenAPI spec is empty")
	ErrInvalidHTTPVerb        = errors.New("invalid HTTP verb")
	ErrInvalidURLResource     = errors.New("invalid URL resource")
	ErrCreatingDirectories    = errors.New("error creating directories")
	ErrCreatingFile           = errors.New("error creating file")
)
