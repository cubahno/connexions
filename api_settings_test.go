package connexions

import (
	"encoding/json"
	assert2 "github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsHandler(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	err = CreateSettingsRoutes(router)
	assert.Nil(err)

	t.Run("get", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/.settings", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check the response status code
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}

		assert.Equal("application/x-yaml", w.Header().Get("Content-Type"))
		assert.Greater(w.Body.Len(), 0)
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

		var response map[string]interface{} // You can define a suitable struct for your JSON structure
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode JSON response: %v", err)
		}

		assert.Equal(true, response["success"])
		assert.Equal("Settings restored and reloaded!", response["message"])

		contents, err := os.ReadFile(filePath)
		assert.Nil(err)
		assert.Greater(len(contents), 0)
	})
}
