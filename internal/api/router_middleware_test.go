package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"runtime"
	"testing"
	"time"

	"github.com/cubahno/connexions/internal/config"
	assert2 "github.com/stretchr/testify/assert"
)

func createPlugin(t *testing.T, fn string) *plugin.Plugin {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "foo.go")
	_ = os.WriteFile(filePath, []byte(fn), 0644)

	soName := "userlib.so"
	pluginPath := filepath.Join(dir, soName)

	// Build the user code into a shared library
	// Change working directory to the temporary directory
	cmdArgs := []string{"build", "-buildmode=plugin"}

	// Check if the environment variable is set
	if os.Getenv("DEBUG_BUILD") == "true" {
		cmdArgs = append(cmdArgs, "-gcflags", "all=-N -l")
	}

	cmdArgs = append(cmdArgs, "-o", pluginPath, dir)

	cmd := exec.Command("go", cmdArgs...)
	cmd.Env = append(os.Environ(),
		"GOROOT="+runtime.GOROOT(),
		"GOARCH="+runtime.GOARCH,
		"GOOS="+runtime.GOOS,
		"CGO_ENABLED=1",
		"GO111MODULE=on",
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		t.Errorf("Error building plugin: %v", err)
		t.FailNow()
	}

	p, err := plugin.Open(pluginPath)
	if err != nil {
		t.Errorf("Error opening plugin: %v", err)
		t.FailNow()
	}
	return p
}

func TestConditionalLoggingMiddleware(t *testing.T) {
	assert := assert2.New(t)

	t.Run("on", func(t *testing.T) {
		current := os.Getenv("DISABLE_LOGGER")
		defer func() {
			_ = os.Setenv("DISABLE_LOGGER", current)
		}()
		_ = os.Setenv("DISABLE_LOGGER", "false")
		cfg := &config.Config{
			App: config.NewDefaultAppConfig(t.TempDir()),
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hallo, welt!"))
		})

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		f := ConditionalLoggingMiddleware(cfg)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hallo, welt!", string(w.buf))
	})
}

func TestCreateRequestTransformerMiddleware(t *testing.T) {
	assert := assert2.New(t)

	t.Run("request can be successfully transformed", func(t *testing.T) {
		t.Skip("TODO:")
		p := createPlugin(t, `package main

import "net/http"

func Foo(resource string, request *http.Request) (*http.Request, error){
	res := request.Clone(request.Context())

	newURL := request.URL
	newURL.Path = "/bar"
	res.URL = newURL

	return res, nil
}
`)
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/bar" {
				t.Errorf("Expected request URL to be '/bar', but got '%s'", r.URL.Path)
			}
			_, _ = w.Write([]byte("Hallo, Welt!"))
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		params := &MiddlewareParams{
			ServiceConfig: &config.ServiceConfig{
				RequestTransformer: "Foo",
			},
			Service:  "Foo",
			Resource: "/bar",
			Plugin:   p,
		}
		f := CreateRequestTransformerMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hallo, Welt!", w.Body.String())
	})
}

func TestCreateUpstreamRequestMiddleware(t *testing.T) {
	assert := assert2.New(t)

	t.Run("upstream service response is used if present", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message": "Hallo, Motto!"}`))
		}))

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hallo, welt!"))
		})

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo", nil)
		history := NewCurrentRequestStorage(100 * time.Millisecond)

		params := &MiddlewareParams{
			ServiceConfig: &config.ServiceConfig{
				Upstream: &config.UpstreamConfig{
					URL: testServer.URL,
				},
			},
			history: history,
		}
		f := CreateUpstreamRequestMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal(`{"message": "Hallo, Motto!"}`, string(w.buf))

		data := history.getData()
		assert.Equal(1, len(data))
		rec := data["GET:/foo"]
		assert.Equal(200, rec.Response.StatusCode)
		assert.Equal([]byte(`{"message": "Hallo, Motto!"}`), rec.Response.Data)
	})
}

func TestCreateResponseMiddleware(t *testing.T) {
	assert := assert2.New(t)

	t.Run("request can be successfully transformed", func(t *testing.T) {
		t.Skip("TODO")
		p := createPlugin(t, `package main

import "github.com/cubahno/pkg/plugin"

func Foo(resource *plugin.RequestedResource) ([]byte, error){
	return []byte("Hallo, Motto!"), nil
}
`)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hallo, Welt!"))
		})

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo", nil)

		history := NewCurrentRequestStorage(100 * time.Millisecond)
		history.Set("foo", req, nil)

		params := &MiddlewareParams{
			ServiceConfig: &config.ServiceConfig{
				ResponseTransformer: "Foo",
			},
			Service:  "Foo",
			Resource: "/foo",
			Plugin:   p,
			history:  history,
		}
		f := CreateResponseMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hallo, Motto!", string(w.buf))
		// old response not overwritten
		assert.Equal("Hallo, Welt!", string(history.data["GET:/foo"].Response.Data))
	})
}
