package api

import "errors"

var (
	ErrResourceNotFound       = errors.New("resource not found")
	ErrResourceMethodNotFound = errors.New("resource method not found")
)
