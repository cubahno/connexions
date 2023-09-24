//go:build !integration

package connexions

import (
	"errors"
	"fmt"
	assert2 "github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewApp(t *testing.T) {
	assert := assert2.New(t)

	t.Run("no-file-config", func(t *testing.T) {
		cfg := MustConfig("/app")
		app := NewApp(cfg)
		assert.NotNil(app)
		assert.Equal(0, len(app.Router.services))
	})

	t.Run("no-file-config-with-pre-create", func(t *testing.T) {
		cfg := MustConfig("/app")
		cfg.App.CreateFileStructure = true

		defer func() {
			if r := recover(); r != nil {
				err, _ := r.(error)
				if err.Error() != "mkdir /app/resources: no such file or directory" {
					t.Errorf("Unexpected panic value: %v", r)
				}
			} else {
				t.Errorf("Expected a panic, but there was none")
			}
		}()

		_ = NewApp(cfg)
	})

	t.Run("existing-dir-with-pre-create", func(t *testing.T) {
		dir := t.TempDir()
		cfg := MustConfig(dir)
		cfg.App.CreateFileStructure = true

		app := NewApp(cfg)
		assert.NotNil(app)
		assert.Equal(0, len(app.Router.services))
	})
}

func TestMustFileStructure(t *testing.T) {
	assert := assert2.New(t)

	t.Run("dirs-exist", func(t *testing.T) {
		dir := t.TempDir()
		paths := NewPaths(dir)

		_, _ = os.Create(paths.Resources)
		_, _ = os.Create(paths.Samples)
		_, _ = os.Create(paths.Services)
		_, _ = os.Create(paths.Contexts)

		err := MustFileStructure(paths)
		assert.NoError(err)
	})
}

func TestApp_AddBluePrint(t *testing.T) {
	assert := assert2.New(t)

	dir := t.TempDir()
	cfg := MustConfig(dir)
	cfg.EnsureConfigValues()
	cfg.App.DisableUI = true

	app := NewApp(cfg)
	router := app.Router

	t.Run("with-error", func(t *testing.T) {
		bp := func(router *Router) error {
			return errors.New("some error")
		}
		err := app.AddBluePrint(bp)
		assert.NotNil(err)
	})

	t.Run("overwrites", func(t *testing.T) {
		// status-quo: no routes
		assert.Equal(0, len(router.Routes()))

		err := CopyFile(
			filepath.Join("test_fixtures", "document-petstore.yml"),
			filepath.Join(cfg.App.Paths.ServicesOpenAPI, "pets", "index.yml"))
		assert.Nil(err)

		// load petstore document
		err = app.AddBluePrint(loadServices)
		if err != nil {
			t.FailNow()
		}
		assert.Equal(4, len(router.services["pets"].Routes))

		bp := func(router *Router) error {
			router.Get("/pets", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("Hallo, pets!"))
			})
			return nil
		}
		_ = app.AddBluePrint(bp)

		req := httptest.NewRequest(http.MethodGet, "/pets", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(http.StatusOK, w.Code)
		assert.Equal("Hallo, pets!", w.Body.String())
	})
}

func TestApp_Run_panics(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := MustConfig(dir)
	cfg.EnsureConfigValues()
	cfg.App.DisableUI = true
	cfg.App.Port = 80

	app := NewApp(cfg)

	var wg sync.WaitGroup

	// Start the HTTP server in a goroutine.
	wg.Add(2)
	defer wg.Done()
	go func() {
		defer func() {
			wg.Done()
			if r := recover(); r != nil {
				err, _ := r.(error)
				if err.Error() != "listen tcp :80: bind: permission denied" {
					t.Errorf("Unexpected panic value: %v", r)
				}
			} else {
				t.Errorf("Expected a panic, but there was none")
			}
		}()
		app.Run()
	}()

	// give some time app to start
	time.Sleep(1 * time.Second)
}

func TestApp_Run(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := MustConfig(dir)
	cfg.EnsureConfigValues()
	cfg.App.DisableUI = true
	cfg.App.Port = 22333

	app := NewApp(cfg)

	var wg sync.WaitGroup

	// Start the HTTP server in a goroutine.
	wg.Add(2)
	defer wg.Done()
	go func() {
		defer func() {
			wg.Done()
			if r := recover(); r != nil {
				t.Errorf("Expected no panic, but there was one")
			}
		}()
		app.Run()
	}()

	// give some time app to start
	time.Sleep(1 * time.Second)

	bp := func(router *Router) error {
		router.Get("/pets", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Hallo, pets!"))
		})
		return nil
	}
	_ = app.AddBluePrint(bp)

	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/pets", cfg.App.Port))
	if err != nil {
		t.Fatalf("Error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	expected := "Hallo, pets!"

	buffer, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	if string(buffer) != expected {
		t.Errorf("Expected body %s, got %s", expected, string(buffer))
	}
}
