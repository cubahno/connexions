package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cubahno/xs"
	"gopkg.in/yaml.v3"
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
	ServiceName string
	IsOpenAPI   bool
	Method      string
	Resource    string
	FilePath    string
	FileName    string
	Extension   string
	ContentType string
}

type UploadedFile struct {
	Content   []byte
	Filename  string
	Extension string
	Size      int64
}

func GetRequestFile(r *http.Request, name string) (*UploadedFile, error) {
	// Get the uploaded file
	file, handler, _ := r.FormFile(name)
	if file != nil {
		defer file.Close()
	}

	if handler == nil {
		return nil, nil
	}

	file, err := handler.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buff := bytes.NewBuffer(make([]byte, 0))
	part := make([]byte, 1024)
	count := 0

	for {
		if count, err = reader.Read(part); err != nil {
			break
		}
		buff.Write(part[:count])
	}

	if err != io.EOF {
		return nil, err
	}

	return &UploadedFile{
		Content:   buff.Bytes(),
		Filename:  handler.Filename,
		Extension: filepath.Ext(handler.Filename),
		Size:      handler.Size,
	}, nil
}

func GetPropertiesFromFilePath(filePath string) *FileProperties {
	fileName := path.Base(filePath)
	ext := strings.ToLower(filepath.Ext(fileName))
	isOpenAPI := false

	s := strings.TrimPrefix(strings.Replace(filePath, xs.ServicePath, "", 1), "/")
	parts := strings.Split(s, "/")
	serviceName := parts[0]
	if serviceName == ".openapi" {
		isOpenAPI = true
		if len(parts) > 1 {
			parts = parts[1:]
			// serviceName = parts[0]
		}
		serviceName = strings.TrimSuffix(fileName, ext)
	}

	method := http.MethodGet
	resource := ""

	if len(parts) == 1 {
		resource = fmt.Sprintf("/%s", parts[0])
	}

	if len(parts) > 1 {
		method_ := strings.ToUpper(parts[1])
		if xs.IsValidHTTPVerb(method_) {
			method = method_
		}
		resource = fmt.Sprintf("/%s/%s", serviceName, strings.Join(parts[2:], "/"))
	}

	return &FileProperties{
		ServiceName: serviceName,
		IsOpenAPI:   isOpenAPI,
		Method:      method,
		Resource:    resource,
		FilePath:    filePath,
		FileName:    fileName,
		Extension:   ext,
		ContentType: mime.TypeByExtension(ext),
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

func IsJsonType(content []byte) bool {
	var jsonData map[string]interface{}
	if err := json.Unmarshal(content, &jsonData); err == nil {
		return true
	}
	return false
}

func IsYamlType(content []byte) bool {
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(content, &yamlData); err == nil {
		fmt.Println("Content is YAML")
		return true
	}
	return false
}
