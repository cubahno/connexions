package xs

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FileProperties contains inferred properties of a file that is being loaded from service directory.
type FileProperties struct {
	ServiceName          string
	IsOpenAPI            bool
	Method               string
	Prefix               string
	Resource             string
	FilePath             string
	FileName             string
	Extension            string
	ContentType          string
	Spec                 *Document            `json:"-"`
	ValueReplacerFactory ValueReplacerFactory `json:"-"`
}

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
	} else {
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

func GetPropertiesFromFilePath(filePath string) (*FileProperties, error) {
	fileName := path.Base(filePath)
	ext := strings.ToLower(filepath.Ext(fileName))
	contentType := mime.TypeByExtension(ext)
	resource := ""

	s := strings.TrimPrefix(strings.Replace(filePath, ServicePath, "", 1), "/")
	parts := strings.Split(s, "/")
	serviceName := parts[0]

	if serviceName == RootOpenAPIName {
		parts = parts[1:]
		serviceName = parts[0]
		prefix := ""

		// root level service
		if len(parts) == 1 {
			serviceName = ""
			prefix = ""
		} else {
			// if dirs are present, service name is the first dir
			serviceName = parts[0]
			prefix = "/" + strings.Join(parts, "/")
			prefix = strings.TrimSuffix(prefix, "/"+fileName)
			prefix = strings.TrimSuffix(prefix, "/")
		}

		doc, err := NewDocumentFromFile(filePath)
		if err != nil {
			return nil, err
		}

		if len(doc.Paths) == 0 {
			return nil, ErrNoPathsInSchema
		}

		return &FileProperties{
			ServiceName:          serviceName,
			Prefix:               prefix,
			IsOpenAPI:            true,
			FilePath:             filePath,
			FileName:             fileName,
			Extension:            ext,
			ContentType:          contentType,
			Spec:                 doc,
			ValueReplacerFactory: CreateValueReplacerFactory(),
		}, nil
	}

	method := http.MethodGet
	prefix := ""

	if serviceName == RootServiceName {
		parts = parts[1:]
		serviceName = parts[0]
		prefix = ""

		// root service
		if IsValidHTTPVerb(serviceName) {
			method = strings.ToUpper(serviceName)
			serviceName = ""
			parts = parts[1:]
		}
	}

	// remove from resource parts
	if fileName == "index.json" {
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
		if IsValidHTTPVerb(method_) {
			method = method_
			parts = SliceDeleteAtIndex[string](parts, 1)
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
		ServiceName:          serviceName,
		Prefix:               prefix,
		Method:               method,
		Resource:             resource,
		FilePath:             filePath,
		FileName:             fileName,
		Extension:            ext,
		ContentType:          contentType,
		ValueReplacerFactory: CreateValueReplacerFactory(),
	}, nil
}

func ComposeFileSavePath(service, method, resource, ext string, isOpenAPI bool) string {
	if isOpenAPI {
		return ComposeOpenAPISavePath(service, resource, ext)
	}

	resource = strings.Trim(resource, "/")
	parts := strings.Split(resource, "/")

	res := ServicePath

	if service == "" && len(parts) > 1 {
		service = parts[0]
		parts = parts[1:]
	}

	if service != "" {
		res += "/" + service
	}

	if service == "" && len(parts) == 1 {
		res += "/" + RootServiceName
	}

	if method == "" {
		method = http.MethodGet
	}

	res += "/" + strings.ToLower(method)
	res += "/" + strings.Join(parts, "/")
	res = strings.TrimSuffix(res, "/")

	pathExt := filepath.Ext(res)
	if pathExt == "" {
		res += "/index" + ext
		if ext == "" {
			res += ".txt"
		}
	}

	return res
}

func ComposeOpenAPISavePath(service, resource, ext string) string {
	resource = strings.Trim(resource, "/")
	parts := strings.Split(resource, "/")

	res := ServiceOpenAPIPath

	if service == "" && len(parts) > 0 {
		service = parts[0]
		parts = parts[1:]
	}

	if service != "" {
		res += "/" + service
	}

	resPart := "/" + strings.Join(parts, "/")
	resPart = strings.TrimSuffix(resPart, "/")

	if resPart == "" {
		resPart = "/index"
	}
	res += resPart + ext

	return res
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

func CopyDirectory(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)
		if info.IsDir() {
			return os.MkdirAll(destPath, os.ModePerm)
		}

		return CopyFile(path, destPath)
	})
}

func CleanupFileStructure() error {
	fmt.Println("Cleaning up file structure...")
	return filepath.WalkDir(ServicePath, func(path string, info os.DirEntry, err error) error {
		// Remove empty directories
		if !info.IsDir() {
			return nil
		}

		isEmpty, err := IsEmptyDir(path)
		if err != nil {
			return nil
		}
		if isEmpty {
			if err := os.Remove(path); err != nil {
				log.Printf("Error removing empty directory: %s\n", err.Error())
				return nil
			}
			log.Printf("Removed empty directory: %s\n", path)
			return nil
		}

		return nil
	})
}

func IsEmptyDir(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
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
