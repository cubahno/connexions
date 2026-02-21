// Package zero This file is generated ONCE as a starting point and will NOT be overwritten.
// Modify it freely to add your business logic.
// To regenerate, delete this file or set generate.handler.output.overwrite: true in config.
package zero

import (
	"github.com/cubahno/connexions/v2/pkg/db"
)

// service implements the ServiceInterface with your business logic.
// Return nil, nil to fall back to the generator for mock responses.
// Return a response to override the generated response.
// Return an error to return an error response.
type service struct {
	db db.DB
}

// Ensure service implements ServiceInterface.
var _ ServiceInterface = (*service)(nil)

// newService creates a new service instance.
// Add your custom initialization logic here.
func newService(serviceDB db.DB) *service {
	return &service{db: serviceDB}
}
