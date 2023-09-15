package connexions

import (
    assert2 "github.com/stretchr/testify/assert"
    "os"
    "path/filepath"
    "testing"
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
    err = CopyFile(filepath.Join("test_fixtures", "fixed-petstore-post-pets.json"), fixedFilePath)
    assert.Nil(err)
    fixedFileProps, err := GetPropertiesFromFilePath(fixedFilePath, router.Config.App)
    assert.Nil(err)

    // copy openapi resource
    openAPIfilePath := filepath.Join(router.Config.App.Paths.ServicesOpenAPI, "ps-openapi", "index.yml")
    err = CopyFile(filepath.Join("test_fixtures", "document-pet-single.yml"), openAPIfilePath)
    assert.Nil(err)
    openAPIFileProps, err := GetPropertiesFromFilePath(openAPIfilePath, router.Config.App)
    assert.Nil(err)

    err = loadServices(router)
    assert.Nil(err)

    assert.Equal(2, len(router.Services))
    assert.NotNil(router.Services[fixedFileProps.ServiceName])
    assert.NotNil(router.Services[openAPIFileProps.ServiceName])
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
    err = CopyFile(filepath.Join("test_fixtures", "fixed-petstore-post-pets.json"), fixedFilePath)
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
    err = CopyFile(filepath.Join("test_fixtures", "document-pet-single.yml"), openAPIfilePath)
    assert.Nil(err)

    _ = os.Chmod(openAPIfilePath, 0000)

    err = loadServices(router)
    // error not returned, but file is skipped
    assert.Nil(err)
    assert.Equal(0, len(router.Services))

    // restore
    _ = os.Chmod(openAPIfilePath, 0777)
}

func TestLoadContexts(t *testing.T) {

}
