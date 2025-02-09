//go:build !integration

package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cubahno/connexions/internal/openapi"
	"github.com/cubahno/connexions/internal/types"
	"github.com/cubahno/connexions_plugin"
	assert2 "github.com/stretchr/testify/assert"
)

func TestLoadServices(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	// prepare files
	// copy fixed resource
	fixedFilePath := filepath.Join(router.Config.App.Paths.Services, "ps-fixed", "post", "pets", "index.json")
	err = types.CopyFile(filepath.Join(testDataPath, "fixed-petstore-post-pets.json"), fixedFilePath)
	assert.Nil(err)
	fixedFileProps, err := openapi.GetPropertiesFromFilePath(fixedFilePath, router.Config.App)
	assert.Nil(err)

	// copy openapi resource
	openAPIfilePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "ps-openapi", "index.yml")
	err = types.CopyFile(filepath.Join(testDataPath, "document-pet-single.yml"), openAPIfilePath)
	assert.Nil(err)
	openAPIFileProps, err := openapi.GetPropertiesFromFilePath(openAPIfilePath, router.Config.App)
	assert.Nil(err)

	err = loadServices(router)
	assert.Nil(err)

	assert.Equal(2, len(router.services))
	assert.NotNil(router.services[fixedFileProps.ServiceName])
	assert.NotNil(router.services[openAPIFileProps.ServiceName])
}

func TestLoadServices_errorReadingDir(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	// prepare files
	fixedFilePath := filepath.Join(router.Config.App.Paths.Services, "ps-fixed", "post", "pets", "index.json")
	err = types.CopyFile(filepath.Join(testDataPath, "fixed-petstore-post-pets.json"), fixedFilePath)
	assert.Nil(err)

	_ = os.Chmod(router.Config.App.Paths.Services, 0000)

	err = loadServices(router)
	assert.Error(err)

	// restore
	_ = os.Chmod(router.Config.App.Paths.Services, 0777)
}

func TestLoadServices_errorGettingFileProps(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	// prepare files
	openAPIfilePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "ps-openapi", "index.yml")
	err = types.CopyFile(filepath.Join(testDataPath, "document-pet-single.yml"), openAPIfilePath)
	assert.Nil(err)

	_ = os.Chmod(openAPIfilePath, 0000)

	err = loadServices(router)
	// error not returned, but file is skipped
	assert.Nil(err)
	assert.Equal(0, len(router.services))

	// restore
	_ = os.Chmod(openAPIfilePath, 0777)
}

func TestLoadContexts(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	files := []string{
		filepath.Join(testDataPath, "context-common.yml"),
		filepath.Join(testDataPath, "context-petstore.yml"),
		filepath.Join(testDataPath, "context-invalid.yml"),
	}

	// copy files
	for _, file := range files {
		err = types.CopyFile(file, filepath.Join(router.Config.App.Paths.Contexts, filepath.Base(file)))
		assert.Nil(err)
	}

	// no error to evaluate
	_ = loadContexts(router)

	res := router.contexts
	assert.Equal(2, len(res))
}

func TestLoadCallbacks(t *testing.T) {
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	filePath := filepath.Join(router.Config.App.Paths.Callbacks, "foo.go")
	if err = types.CopyFile(filepath.Join(testDataPath, "callbacks", "foo.go"), filePath); err != nil {
		t.Errorf("Error copying file: %v", err)
		t.FailNow()
	}

	if err = loadCallbacks(router); err != nil {
		t.Errorf("Error loading callbacks: %v", err)
		t.FailNow()
	}

	symbol, err := router.callbacksPlugin.Lookup("Foo")
	if err != nil {
		t.Errorf("Error looking up symbol: %v", err)
		t.FailNow()
	}

	_, ok := symbol.(func(*connexions_plugin.RequestedResource) ([]byte, error))
	assert.True(ok)
}
