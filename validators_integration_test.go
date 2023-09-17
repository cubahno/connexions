//go:build integration

package connexions

import (
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type validationResult struct {
	file    string
	path    string
	method  string
	ok      bool
	docErr  string
	reqErr  string
	respErr string
}

// TestValidateResponse_Integration is an end-to-end test that validates the response of the API.
// This test is skipped by default. To run this test, use make test-integration.
// Specs should be located in resources/specs, they all .gitignored except petstore.yml.
func TestValidateResponse_Integration(t *testing.T) {
	wg := &sync.WaitGroup{}
	ch := make(chan validationResult, 0)
	specDir := filepath.Join("resources", "test", "specs")
	replacerFactory := CreateValueReplacerFactory(Replacers)

	_ = filepath.Walk(specDir, func(filePath string, info os.FileInfo, fileErr error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		wg.Add(1)

		go func(filePath string) {
			defer wg.Done()
			base := filepath.Base(filePath)
			service := strings.TrimSuffix(base, filepath.Ext(base))
			replacer := replacerFactory(&Resource{
				Service:     service,
				ContextData: []map[string]any{},
			})

			validateFile(filePath, replacer, ch)
		}(filePath)

		return nil
	})

	go func() {
		wg.Wait()
		close(ch)
	}()

	failsDescr := make([]validationResult, 0)
	success := 0
	fails := 0

	for res := range ch {
		if res.ok {
			success++
			continue
		}
		failsDescr = append(failsDescr, res)
		fails++
	}

	fmt.Pprintf("Success: %d, Fails: %d\n", success, fails)
	if fails > 0 {
		t.Errorf("Failed to validate %d resources", fails)
		for _, res := range failsDescr {
			if res.docErr != "" {
				t.Errorf("Document error in file: %s\n%s\n\n", res.file, res.docErr)
				continue
			}
			if res.reqErr != "" || res.respErr != "" {
				t.Errorf("File: %s\nPath: %s\nMethod: %s\n", res.file, res.path, res.method)
				if res.reqErr != "" {
					fmt.Printf("Request error: %s\n", res.reqErr)
				}
				if res.respErr != "" {
					fmt.Printf("Response error: %s\n", res.respErr)
				}
			}

			println()
		}
	}
}

func validateFile(filePath string, replacer ValueReplacer, ch chan<- validationResult) {
	fileName := filepath.Base(filePath)
	// there should be a simple way to tmp skip some specs
	if fileName[0] == '-' {
		return
	}

	doc, err := NewDocumentFromFileFactory(KinOpenAPIProvider)(filePath)
	if err != nil {
		ch <- validationResult{
			file:   fileName,
			docErr: err.Error(),
		}
		return
	}

	for resource, methods := range doc.GetResources() {
		for _, method := range methods {
			println(fmt.Sprintf("Validating %s %s", method, resource))
			operation := doc.FindOperation(&FindOperationOptions{
				Resource: resource,
				Method:   method,
			})

			reqBody, reqContentType := operation.GetRequestBody()
			req := NewRequestFromOperation("", resource, method, operation, replacer)

			var body io.ReadCloser
			if reqBody != nil {
				body = io.NopCloser(strings.NewReader(req.Body))
			}

			request := httptest.NewRequest(method, resource, body)
			request.Header.Set("Content-Type", reqContentType)

			success := false

			reqErr := ValidateRequest(request, reqBody, reqContentType)
			reqErrMsg := ""
			if reqErr != nil {
				reqErrMsg = reqErr.Error()
			}

			respErrMsg := ""
			response := NewResponseFromOperation(operation, replacer)
			respErr := ValidateResponse(request, response, operation)
			if respErr != nil {
				respErrMsg = respErr.Error()
			}

			if respErr == nil && reqErr == nil {
				success = true
			}

			ch <- validationResult{
				file:    fileName,
				path:    resource,
				method:  method,
				ok:      success,
				reqErr:  reqErrMsg,
				respErr: respErrMsg,
			}
		}
	}
}
