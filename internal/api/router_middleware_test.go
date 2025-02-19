package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"plugin"
	"testing"
	"time"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/testhelpers"
	assert2 "github.com/stretchr/testify/assert"
)

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

func TestCreateBeforeHandlerMiddleware(t *testing.T) {
	assert := assert2.New(t)

	t.Run("request can be successfully transformed", func(t *testing.T) {
		pluginPath := testhelpers.CreateTestPlugin()
		p, err := plugin.Open(pluginPath)
		if err != nil {
			t.Errorf("Error opening plugin: %v", err)
			t.FailNow()
		}
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
				Middleware: &config.MiddlewareConfig{
					BeforeHandler: []string{"ReplaceRequestURL"},
				},
			},
			Service:  "Foo",
			Resource: "/bar",
			Plugin:   p,
			history:  NewCurrentRequestStorage(100 * time.Millisecond),
		}
		f := CreateBeforeHandlerMiddleware(params)
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
		pluginPath := testhelpers.CreateTestPlugin()
		p, err := plugin.Open(pluginPath)
		if err != nil {
			t.Errorf("Error opening plugin: %v", err)
			t.FailNow()
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Hallo, Welt!"))
		})

		w := NewBufferedResponseWriter()
		req := httptest.NewRequest(http.MethodGet, "/foo", nil)

		history := NewCurrentRequestStorage(100 * time.Millisecond)
		history.Set("foo", "foo", req, nil)

		params := &MiddlewareParams{
			ServiceConfig: &config.ServiceConfig{
				Middleware: &config.MiddlewareConfig{
					AfterHandler: []string{"ReplaceResponse"},
				},
			},
			Service:  "Foo",
			Resource: "/foo",
			Plugin:   p,
			history:  history,
		}
		f := CreateAfterHandlerMiddleware(params)
		f(handler).ServeHTTP(w, req)

		assert.Equal("Hallo, Motto!", string(w.buf))
		// old response overwritten
		assert.Equal("Hallo, Motto!", string(history.data["GET:/foo"].Response.Data))
	})
}
