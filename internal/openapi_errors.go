package internal

import "errors"

var (
	ErrUnexpectedFormDataType       = errors.New("expected map[string]any for multipart/form-data")
	ErrUnexpectedFormURLEncodedType = errors.New("expected map[string]any for x-www-form-urlencoded")
)
