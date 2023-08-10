package xs

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FileProperties contains inferred properties of a file that is being loaded from service directory.
type FileProperties struct {
	ServiceName       string
	IsPossibleOpenAPI bool
	Method            string
	Resource          string
	FilePath          string
	FileName          string
	Extension         string
	ContentType       string
}

func GetPropertiesFromFilePath(filePath string) *FileProperties {
	fileName := path.Base(filePath)
	ext := strings.ToLower(filepath.Ext(fileName))

	s := strings.TrimPrefix(strings.Replace(filePath, ServicePath, "", 1), "/")
	parts := strings.Split(s, "/")
	serviceName := parts[0]
	method := ""
	resource := ""
	isPossibleOpenAPI := false

	if serviceName == fileName {
		serviceName = strings.TrimSuffix(fileName, ext)
		if ext == ".yaml" || ext == ".yml" || ext == ".json" {
			isPossibleOpenAPI = true
		}
	} else if len(parts) > 1 {
		method = strings.ToUpper(parts[1])
		if !IsValidHTTPVerb(method) {
			method = http.MethodGet
		}
		resource = fmt.Sprintf("/%s/%s", serviceName, strings.Join(parts[2:], "/"))
	}

	return &FileProperties{
		ServiceName:       serviceName,
		IsPossibleOpenAPI: isPossibleOpenAPI,
		Method:            method,
		Resource:          resource,
		FilePath:          filePath,
		FileName:          fileName,
		Extension:         ext,
		ContentType:       mime.TypeByExtension(ext),
	}
}

func CopyFile(srcPath, destPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}
