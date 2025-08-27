package openapi

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/types"
	gomime "github.com/cubewise-code/go-mime"
)

// FileProperties contains inferred properties of a file that is being loaded from service directory.
//
// ServiceName is the name of the service that the file belongs to.
// It represents the first directory in the file path.
//
// IsOpenAPI indicates whether the file is an OpenAPI specification.
// Method is the HTTP method of the resource, which this file describes.
// Prefix is the path prefix of the resource, which this file describes.
// This is service name with a leading slash.
//
// Resource is the path of the resource, which this file describes without prefix.
// FilePath is the full path to the file.
// FileName is the name of the file with the extension.
// Extension is the extension of the file, with the leading dot.
// ContentType is the MIME type of the file.
// Spec is the OpenAPI specification of the file if the file iis an OpenAPI specification.
type FileProperties struct {
	ServiceName string
	IsOpenAPI   bool
	Method      string
	Prefix      string
	Resource    string
	FilePath    string
	FileName    string
	Extension   string
	ContentType string
	Spec        Document `json:"-"`
}

// IsEqual compares two FileProperties structs.
// Spec is not compared.
func (f *FileProperties) IsEqual(other *FileProperties) bool {
	return f.ServiceName == other.ServiceName &&
		f.IsOpenAPI == other.IsOpenAPI &&
		f.Method == other.Method &&
		f.Prefix == other.Prefix &&
		f.Resource == other.Resource &&
		f.FilePath == other.FilePath &&
		f.FileName == other.FileName &&
		f.Extension == other.Extension &&
		f.ContentType == other.ContentType
}

// UploadedFile represents an uploaded file.
// Content is the content of the file.
// Filename is the name of the file.
// Extension is the extension of the file with the leading dot.
// Size is the size of the file in bytes.
type UploadedFile struct {
	Content   []byte
	Filename  string
	Extension string
	Size      int64
}

// GetRequestFile gets an uploaded file from a GeneratedRequest.
func GetRequestFile(r *http.Request, fieldName string) (*UploadedFile, error) {
	// Get the uploaded file
	file, header, _ := r.FormFile(fieldName)
	if file != nil {
		defer func() { _ = file.Close() }()
	} else {
		return nil, nil
	}

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, err
	}

	return &UploadedFile{
		Content:   buf.Bytes(),
		Filename:  header.Filename,
		Extension: filepath.Ext(header.Filename),
		Size:      header.Size,
	}, nil
}

// GetPropertiesFromFilePath gets properties of a file from its path.
func GetPropertiesFromFilePath(filePath string, appCfg *config.AppConfig) (*FileProperties, error) {
	s := strings.TrimPrefix(strings.Replace(filePath, appCfg.Paths.Services, "", 1), "/")
	parts := strings.Split(s, "/")
	serviceName := parts[0]

	if serviceName == config.RootOpenAPIName {
		return getPropertiesFromOpenAPIFile(filePath, parts[1:], appCfg)
	}

	return getPropertiesFromFixedFile(serviceName, filePath, parts), nil
}

func getPropertiesFromOpenAPIFile(filePath string, pathParts []string, appCfg *config.AppConfig) (*FileProperties, error) {
	fileName := path.Base(filePath)
	ext := strings.ToLower(filepath.Ext(fileName))

	prefix := ""
	var serviceName string

	// root level service
	if len(pathParts) == 1 {
		serviceName = ""
		prefix = ""
	} else {
		// if dirs are present, service name is the first dir
		serviceName = pathParts[0]
		prefix = "/" + strings.Join(pathParts, "/")
		prefix = strings.TrimSuffix(prefix, "/"+fileName)
		prefix = strings.TrimSuffix(prefix, "/")
	}

	doc, err := NewDocumentFromFile(filePath)
	if err != nil {
		return nil, err
	}

	return &FileProperties{
		ServiceName: serviceName,
		Prefix:      prefix,
		IsOpenAPI:   true,
		FilePath:    filePath,
		FileName:    fileName,
		Extension:   ext,
		// set content type empty, because it will be set by the corresponding resource
		ContentType: "",
		Spec:        doc,
	}, nil
}

func getPropertiesFromFixedFile(serviceName, filePath string, parts []string) *FileProperties {
	fileName := path.Base(filePath)
	ext := strings.ToLower(filepath.Ext(fileName))
	// TODO: fix content type for all platforms
	contentType := gomime.TypeByExtension(ext)
	resource := ""

	method := http.MethodGet
	prefix := ""

	if serviceName == config.RootServiceName {
		parts = parts[1:]
		serviceName = parts[0]

		// root service
		if types.IsValidHTTPVerb(serviceName) {
			method = strings.ToUpper(serviceName)
			parts = parts[1:]
		}
	}

	// remove from resource parts
	if strings.HasPrefix(fileName, "index.") {
		parts = parts[:len(parts)-1]
	}

	// root level service
	if len(parts) == 1 {
		serviceName = ""
		prefix = ""
		resource = fmt.Sprintf("/%s", parts[0])
	} else if len(parts) > 1 {
		serviceName = parts[0]
		method_ := strings.ToUpper(parts[1])
		if types.IsValidHTTPVerb(method_) {
			method = method_
			parts = types.SliceDeleteAtIndex[string](parts, 1)
		}
		resource = fmt.Sprintf("/%s", strings.Join(parts[1:], "/"))
		prefix = "/" + serviceName
	} else {
		serviceName = ""
		prefix = ""
	}

	prefix = strings.TrimSuffix(prefix, "/")

	if resource == "" {
		resource = "/"
	}

	return &FileProperties{
		ServiceName: serviceName,
		Prefix:      prefix,
		Method:      method,
		Resource:    resource,
		FilePath:    filePath,
		FileName:    fileName,
		Extension:   ext,
		ContentType: contentType,
	}
}
