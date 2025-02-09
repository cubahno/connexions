//go:build !integration

package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cubahno/connexions/internal/types"
	assert2 "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)

		assert.Equal(true, resp.Success)
		assert.Equal("Settings saved and reloaded!", resp.Message)
	})

	t.Run("post", func(t *testing.T) {
		// save invalid config
		filePath := router.Config.App.Paths.ConfigFile
		err = types.SaveFile(filePath, []byte(""))
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

		resp := UnmarshallResponse[SimpleResponse](t, w.Body)

		assert.Equal(true, resp.Success)
		assert.Equal("Settings restored and reloaded!", resp.Message)

		contents, err := os.ReadFile(filePath)
		assert.Nil(err)
		assert.Greater(len(contents), 0)
	})
}

func TestSettingsHandler_Put_ErrorWriting(t *testing.T) {
	assert := require.New(t)

	router, err := SetupApp(t.TempDir())
	assert.Nil(err)
	err = createSettingsRoutes(router)
	assert.Nil(err)

	// set as read-only
	err = os.Chmod(router.Config.App.Paths.ConfigFile, 0400)
	assert.NoError(err)

	payload := `
app:
  port: 8080
`
	req := httptest.NewRequest("PUT", "/.settings", strings.NewReader(payload))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := UnmarshallResponse[SimpleResponse](t, w.Body)

	assert.Equal(500, w.Code)
	assert.Equal(false, resp.Success)
	assert.True(strings.HasSuffix(resp.Message, "resources/data/config.yml: permission denied"))
}

func TestSettingsHandler_Put_InvalidYaml(t *testing.T) {
	assert := require.New(t)

	router, err := SetupApp(t.TempDir())
	assert.Nil(err)
	err = createSettingsRoutes(router)
	assert.Nil(err)

	payload := `1`
	req := httptest.NewRequest("PUT", "/.settings", strings.NewReader(payload))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := UnmarshallResponse[SimpleResponse](t, w.Body)

	assert.Equal(400, w.Code)
	assert.Equal(false, resp.Success)
	assert.True(strings.HasPrefix(resp.Message, "yaml: unmarshal errors:"))
}

func TestSettingsHandler_Post_WriteError(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	assert.Nil(err)
	err = createSettingsRoutes(router)
	assert.Nil(err)

	// set as read-only
	err = os.Chmod(router.Config.App.Paths.ConfigFile, 0400)
	assert.NoError(err)

	req := httptest.NewRequest("POST", "/.settings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := UnmarshallResponse[SimpleResponse](t, w.Body)

	assert.Equal(500, w.Code)
	assert.Equal(false, resp.Success)
	assert.Equal("Failed to restore config contents", resp.Message)
}
