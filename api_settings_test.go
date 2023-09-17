//go:build !integration

package connexions

import (
	"encoding/json"
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateSettingsRoutes_Disabled(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()
	router, _ := SetupApp(t.TempDir())
	router.Config.App.DisableUI = true

	_ = createSettingsRoutes(router)
	assert.Equal(0, len(router.Mux.Routes()))
}

func TestSettingsHandler(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = createSettingsRoutes(router)
	assert.Nil(err)

	t.Run("get", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/.settings", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("application/x-yaml", w.Header().Get("Content-Type"))
		assert.Greater(w.Body.Len(), 0)
	})

	t.Run("put", func(t *testing.T) {
		payload := `
app:
  port: 8080
`
		req := httptest.NewRequest("PUT", "/.settings", strings.NewReader(payload))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal("application/json", w.Header().Get("Content-Type"))

		assert.Equal(8080, router.Config.App.Port)
		var response map[string]interface{}
		if err = json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}
		assert.Equal(true, response["success"])
		assert.Equal("Settings saved and reloaded!", response["message"])
	})

	t.Run("put-no-body", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/.settings", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var response map[string]any
		_ = json.NewDecoder(w.Body).Decode(&response)

		assert.Equal(400, w.Code)
		assert.Equal(false, response["success"])
		assert.Equal("invalid config", response["message"])
	})

	t.Run("post", func(t *testing.T) {
		// save invalid config
		filePath := filepath.Join(router.Config.App.Paths.Resources, "config.yml")
		err = SaveFile(filePath, []byte(""))
		assert.Nil(err)

		// now restore it
		req := httptest.NewRequest("POST", "/.settings", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check the response status code
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}

		assert.Equal("application/json", w.Header().Get("Content-Type"))

		var response map[string]any
		_ = json.NewDecoder(w.Body).Decode(&response)

		assert.Equal(true, response["success"])
		assert.Equal("Settings restored and reloaded!", response["message"])

		contents, err := os.ReadFile(filePath)
		assert.Nil(err)
		assert.Greater(len(contents), 0)
	})
}

func TestSettingsHandler_Put_ErrorWriting(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	assert.Nil(err)
	err = createSettingsRoutes(router)
	assert.Nil(err)

	// set as read-only
	err = os.Chmod(router.Config.App.Paths.ConfigFile, 0400)
	assert.Nil(err)

	payload := `
app:
  port: 8080
`
	req := httptest.NewRequest("PUT", "/.settings", strings.NewReader(payload))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]any
	_ = json.NewDecoder(w.Body).Decode(&response)

	assert.Equal(500, w.Code)
	assert.Equal(false, response["success"])
	assert.True(strings.HasSuffix(response["message"].(string), "resources/config.yml: permission denied"))
}

func TestSettingsHandler_Post_ErrorCopyFile(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	assert.Nil(err)
	err = createSettingsRoutes(router)
	assert.Nil(err)

	router.Config.App.Paths.ConfigFile = "non-existent.yml"

	req := httptest.NewRequest("POST", "/.settings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response map[string]any
	_ = json.NewDecoder(w.Body).Decode(&response)

	assert.Equal(500, w.Code)
	assert.Equal(false, response["success"])
	assert.Equal("Failed to copy file contents", response["message"])
}
