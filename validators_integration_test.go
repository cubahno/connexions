//go:build integration

package connexions

import (
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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
	filePath := os.Getenv("SPEC")

	maxFailsVal := os.Getenv("MAX_FAILS")
	maxFails := 0
	if maxFailsVal != "" {
		maxFails, _ = strconv.Atoi(maxFailsVal)
	}
	if maxFails == 0 {
		maxFails = 5
	}

	println("Validating specs...", filePath)

	wg := &sync.WaitGroup{}
	ch := make(chan validationResult)
	stopCh := make(chan struct{})

	specDir := filepath.Join("resources", "test", "specs")
	cfg := NewDefaultConfig("")

	if filePath != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			valueReplacer := CreateValueReplacer(cfg, nil)
			validateFile(filePath, valueReplacer, ch, stopCh)
		}()
	} else {
		_ = filepath.Walk(specDir, func(filePath string, info os.FileInfo, fileErr error) error {
			if info == nil || info.IsDir() {
				return nil
			}

			wg.Add(1)

			go func(filePath string) {
				defer wg.Done()
				valueReplacer := CreateValueReplacer(cfg, nil)
				validateFile(filePath, valueReplacer, ch, stopCh)
			}(filePath)

			return nil
		})
	}

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

		if fails >= maxFails {
			fmt.Printf("Max fails reached: %d\n", maxFails)
			close(stopCh)
			break
		}
	}

	fmt.Printf("Success: %d, Fails: %d\n", success, fails)
	if fails > 0 {
		t.Errorf("Failed to validate %d resources", fails)
		for i, res := range failsDescr {
			fmt.Printf("Fail %d:\n====================\n", i+1)
			if res.docErr != "" {
				fmt.Printf("Document error in file: %s\n%s\n\n", res.file, res.docErr)
				continue
			}
			if res.reqErr != "" || res.respErr != "" {
				fmt.Printf("File: %s\nPath: %s\nMethod: %s\n", res.file, res.path, res.method)
				if res.reqErr != "" {
					fmt.Printf("Request error: %s\n", res.reqErr)
				}
				if res.respErr != "" {
					fmt.Printf("Response error: %s\n", res.respErr)
				}
			}

			fmt.Printf("==========================\n\n")
		}
	}
}

func validateFile(filePath string, replacer ValueReplacer, ch chan<- validationResult, stop <-chan struct{}) {
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
	validator := NewOpenAPIValidator(doc)
	if validator == nil {
		ch <- validationResult{
			file:   fileName,
			docErr: "Failed to create validator",
		}
		return
	}

	for resource, methods := range doc.GetResources() {
		for _, method := range methods {
			println(fmt.Sprintf("Validating [%s]: %s %s", fileName, method, resource))
			operation := doc.FindOperation(&OperationDescription{
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

			reqErrs := validator.ValidateRequest(&Request{
				Headers:     request.Header,
				Method:      request.Method,
				Path:        request.URL.Path,
				ContentType: reqContentType,
				operation:   operation,
				request:     request,
			})
			reqErrMsg := ""
			if len(reqErrs) > 0 {
				reqErrMsg = fmt.Sprintf("Request validation failed: %d errors\n", len(reqErrs))
				for _, reqErr := range reqErrs {
					reqErrMsg += reqErr.Error() + "\n"
				}
			}

			respErrMsg := ""
			response := NewResponseFromOperation(request, operation, replacer)
			respErrs := validator.ValidateResponse(response)

			if len(respErrs) > 0 {
				respErrMsg = fmt.Sprintf("Response validation failed: %d errors\n", len(respErrs))
				for _, respErr := range respErrs {
					respErrMsg += respErr.Error() + "\n"
				}
			}

			if respErrMsg == "" && reqErrMsg == "" {
				success = true
			}

			select {
			case <-stop:
				return
			default:
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
}
