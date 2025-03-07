//go:build !integration

package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cubahno/connexions/internal/files"
	"github.com/cubahno/connexions/internal/openapi"
	"github.com/cubahno/connexions/internal/testhelpers"
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
	err = files.CopyFile(filepath.Join(testDataPath, "fixed-petstore-post-pets.json"), fixedFilePath)
	assert.Nil(err)
	fixedFileProps, err := openapi.GetPropertiesFromFilePath(fixedFilePath, router.Config.App)
	assert.Nil(err)

	// copy openapi resource
	openAPIfilePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "ps-openapi", "index.yml")
	err = files.CopyFile(filepath.Join(testDataPath, "document-pet-single.yml"), openAPIfilePath)
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
	err = files.CopyFile(filepath.Join(testDataPath, "fixed-petstore-post-pets.json"), fixedFilePath)
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
	err = files.CopyFile(filepath.Join(testDataPath, "document-pet-single.yml"), openAPIfilePath)
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

	flz := []string{
		filepath.Join(testDataPath, "context-common.yml"),
		filepath.Join(testDataPath, "context-petstore.yml"),
		filepath.Join(testDataPath, "context-invalid.yml"),
	}

	// copy files
	for _, file := range flz {
		err = files.CopyFile(file, filepath.Join(router.Config.App.Paths.Contexts, filepath.Base(file)))
		assert.Nil(err)
	}

	// no error to evaluate
	_ = loadContexts(router)

	res := router.contexts
	assert.Equal(2, len(res))
}

func TestLoadPlugins(t *testing.T) {
	t.Skip("TODO:")
	assert := assert2.New(t)

	router, err := SetupApp(t.TempDir())
	if err != nil {
		t.Errorf("Error setting up app: %v", err)
		t.FailNow()
	}

	pluginPath := testhelpers.CreateTestPlugin()
	if err = files.CopyFile(pluginPath, router.Config.App.Paths.Plugins); err != nil {
		t.Errorf("Error copying file: %v", err)
		t.FailNow()
	}

	if err = loadPlugins(router); err != nil {
		t.Errorf("Error loading plugins: %v", err)
		t.FailNow()
	}

	plug := router.middlewarePlugin
	assert.NotNil(plug)
	if plug == nil {
		t.FailNow()
	}

	symbol, err := router.middlewarePlugin.Lookup("Foo")
	if err != nil {
		t.Errorf("Error looking up symbol: %v", err)
		t.FailNow()
	}

	_, ok := symbol.(func(*connexions_plugin.RequestedResource) ([]byte, error))
	assert.True(ok)
}
