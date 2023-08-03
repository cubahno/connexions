package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewJSONResponse(t *testing.T) {
	t.Run("Creates new JSONResponse instance", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewJSONResponse(w)

		assert.NotNil(t, resp)
		assert.NotNil(t, resp.w)
		assert.NotNil(t, resp.headers)
		assert.Equal(t, 0, resp.statusCode)
	})
}

func TestJSONResponse_WithHeader(t *testing.T) {
	t.Run("Adds single header", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewJSONResponse(w)

		resp.WithHeader("X-Custom", "value")

		assert.Equal(t, "value", resp.headers["X-Custom"])
	})

	t.Run("Adds multiple headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewJSONResponse(w)

		resp.WithHeader("X-Custom-1", "value1").
			WithHeader("X-Custom-2", "value2")

		assert.Equal(t, "value1", resp.headers["X-Custom-1"])
		assert.Equal(t, "value2", resp.headers["X-Custom-2"])
	})

	t.Run("Overwrites existing header", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewJSONResponse(w)

		resp.WithHeader("X-Custom", "old").
			WithHeader("X-Custom", "new")

		assert.Equal(t, "new", resp.headers["X-Custom"])
	})
}

func TestJSONResponse_WithStatusCode(t *testing.T) {
	t.Run("Sets status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewJSONResponse(w)

		resp.WithStatusCode(http.StatusCreated)

		assert.Equal(t, http.StatusCreated, resp.statusCode)
	})

	t.Run("Chains with other methods", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := NewJSONResponse(w)

		resp.WithStatusCode(http.StatusNotFound).
			WithHeader("X-Custom", "value")

		assert.Equal(t, http.StatusNotFound, resp.statusCode)
		assert.Equal(t, "value", resp.headers["X-Custom"])
	})
}

func TestJSONResponse_Send(t *testing.T) {
	t.Run("Sends JSON with default status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"message": "hello"}

		NewJSONResponse(w).Send(data)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var result map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.Equal(t, "hello", result["message"])
	})

	t.Run("Sends JSON with custom status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"error": "not found"}

		NewJSONResponse(w).WithStatusCode(http.StatusNotFound).Send(data)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})

	t.Run("Sends JSON with custom headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"data": "test"}

		NewJSONResponse(w).
			WithHeader("X-Custom-Header", "custom-value").
			WithHeader("X-Request-ID", "123").
			Send(data)

		assert.Equal(t, "custom-value", w.Header().Get("X-Custom-Header"))
		assert.Equal(t, "123", w.Header().Get("X-Request-ID"))
	})

	t.Run("Sends nil data as null", func(t *testing.T) {
		w := httptest.NewRecorder()

		NewJSONResponse(w).Send(nil)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "null", w.Body.String())
	})

	t.Run("Handles marshal error gracefully", func(t *testing.T) {
		w := httptest.NewRecorder()
		// channels cannot be marshaled to JSON
		data := make(chan int)

		NewJSONResponse(w).Send(data)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "error")
	})

	t.Run("Sends SimpleResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := &SimpleResponse{
			Success: true,
			Message: "Operation successful",
		}

		NewJSONResponse(w).Send(data)

		assert.Equal(t, http.StatusOK, w.Code)

		var result SimpleResponse
		err := json.Unmarshal(w.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "Operation successful", result.Message)
	})
}

func TestSendHTML(t *testing.T) {
	t.Run("Sends HTML with default status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		html := []byte("<html><body>Hello</body></html>")

		SendHTML(w, 0, html)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
		assert.Equal(t, string(html), w.Body.String())
	})

	t.Run("Sends HTML with custom status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		html := []byte("<html><body>Not Found</body></html>")

		SendHTML(w, http.StatusNotFound, html)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
		assert.Equal(t, string(html), w.Body.String())
	})

	t.Run("Sends empty HTML", func(t *testing.T) {
		w := httptest.NewRecorder()

		SendHTML(w, http.StatusOK, []byte{})

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "", w.Body.String())
	})

	t.Run("Sends HTML with special characters", func(t *testing.T) {
		w := httptest.NewRecorder()
		html := []byte("<html><body>Hello & goodbye <script>alert('test')</script></body></html>")

		SendHTML(w, http.StatusOK, html)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, string(html), w.Body.String())
	})
}

func TestSimpleResponse(t *testing.T) {
	t.Run("Creates success response", func(t *testing.T) {
		resp := &SimpleResponse{
			Success: true,
			Message: "Operation completed",
		}

		assert.True(t, resp.Success)
		assert.Equal(t, "Operation completed", resp.Message)
	})

	t.Run("Creates error response", func(t *testing.T) {
		resp := &SimpleResponse{
			Success: false,
			Message: "Operation failed",
		}

		assert.False(t, resp.Success)
		assert.Equal(t, "Operation failed", resp.Message)
	})

	t.Run("Marshals to JSON correctly", func(t *testing.T) {
		resp := &SimpleResponse{
			Success: true,
			Message: "Test message",
		}

		jsonBytes, err := json.Marshal(resp)
		assert.NoError(t, err)

		var result SimpleResponse
		err = json.Unmarshal(jsonBytes, &result)
		assert.NoError(t, err)
		assert.Equal(t, resp.Success, result.Success)
		assert.Equal(t, resp.Message, result.Message)
	})
}

func TestJSONResponse_Integration(t *testing.T) {
	t.Run("Complete request-response cycle", func(t *testing.T) {
		w := httptest.NewRecorder()

		// Simulate a successful API response
		NewJSONResponse(w).
			WithStatusCode(http.StatusCreated).
			WithHeader("X-Request-ID", "abc123").
			WithHeader("X-Version", "v1").
			Send(&SimpleResponse{
				Success: true,
				Message: "Resource created successfully",
			})

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, "abc123", w.Header().Get("X-Request-ID"))
		assert.Equal(t, "v1", w.Header().Get("X-Version"))

		var result SimpleResponse
		err := json.Unmarshal(w.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "Resource created successfully", result.Message)
	})

	t.Run("Error response with custom status", func(t *testing.T) {
		w := httptest.NewRecorder()

		NewJSONResponse(w).
			WithStatusCode(http.StatusBadRequest).
			Send(&SimpleResponse{
				Success: false,
				Message: "Invalid request parameters",
			})

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var result SimpleResponse
		err := json.Unmarshal(w.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Equal(t, "Invalid request parameters", result.Message)
	})
}
