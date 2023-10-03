package connexions

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/cubahno/connexions/internal"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// FileProperties contains inferred properties of a file that is being loaded from service directory.
type FileProperties struct {
	// ServiceName is the name of the service that the file belongs to.
	// It represents the first directory in the file path.
	ServiceName string

	// IsOpenAPI indicates whether the file is an OpenAPI specification.
	IsOpenAPI bool

	// Method is the HTTP method of the resource, which this file describes.
	Method string

	// Prefix is the path prefix of the resource, which this file describes.
	// This is service name with a leading slash.
	Prefix string

	// Resource is the path of the resource, which this file describes without prefix.
	Resource string

	// FilePath is the full path to the file.
	FilePath string

	// FileName is the name of the file with the extension.
	FileName string

	// Extension is the extension of the file, with the leading dot.
	Extension string

	// ContentType is the MIME type of the file.
	ContentType string

	// Spec is the OpenAPI specification of the file if the file iis an OpenAPI specification.
	Spec Document `json:"-"`
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
type UploadedFile struct {
	// Content is the content of the file.
	Content []byte

	// Filename is the name of the file.
	Filename string

	// Extension is the extension of the file with the leading dot.
	Extension string

	// Size is the size of the file in bytes.
	Size int64
}

// GetRequestFile gets an uploaded file from a request.
func GetRequestFile(r *http.Request, fieldName string) (*UploadedFile, error) {
	// Get the uploaded file
	file, header, _ := r.FormFile(fieldName)
	if file != nil {
		defer file.Close()
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
func GetPropertiesFromFilePath(filePath string, appCfg *AppConfig) (*FileProperties, error) {
	s := strings.TrimPrefix(strings.Replace(filePath, appCfg.Paths.Services, "", 1), "/")
	parts := strings.Split(s, "/")
	serviceName := parts[0]

	if serviceName == RootOpenAPIName {
		return getPropertiesFromOpenAPIFile(filePath, parts[1:], appCfg)
	}

	return getPropertiesFromFixedFile(serviceName, filePath, parts), nil
}

func getPropertiesFromOpenAPIFile(filePath string, pathParts []string, appCfg *AppConfig) (*FileProperties, error) {
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

	doc, err := NewDocumentFromFileFactory(appCfg.SchemaProvider)(filePath)
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
	contentType := mime.TypeByExtension(ext)
	resource := ""

	method := http.MethodGet
	prefix := ""

	if serviceName == RootServiceName {
		parts = parts[1:]
		serviceName = parts[0]

		// root service
		if internal.IsValidHTTPVerb(serviceName) {
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
		if internal.IsValidHTTPVerb(method_) {
			method = method_
			parts = internal.SliceDeleteAtIndex[string](parts, 1)
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

// SaveFile saves a file to the specified path.
// If the destination directory doesn't exist, it will be created.
func SaveFile(filePath string, data []byte) error {
	dirPath := filepath.Dir(filePath)
	// Create directories recursively
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		return err
	}

	dest, err := os.Create(filePath)
	if err != nil {
		return err
	}

	_, err = dest.Write(data)

	return err
}

// CopyFile copies a file from srcPath to destPath.
// If the destination directory doesn't exist, it will be created.
func CopyFile(srcPath, destPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure the destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// CopyDirectory copies a directory recursively.
func CopyDirectory(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return os.MkdirAll(dest, os.ModePerm)
		}

		// compose dest file path, we never get an error here
		relPath, _ := filepath.Rel(src, path)
		destPath := filepath.Join(dest, relPath)
		return CopyFile(path, destPath)
	})
}

// CleanupServiceFileStructure removes empty directories from the service directory.
func CleanupServiceFileStructure(servicePath string) error {
	log.Printf("Cleaning up service file structure %s...\n", servicePath)
	return filepath.WalkDir(servicePath, func(path string, info os.DirEntry, err error) error {
		if !info.IsDir() {
			return nil
		}

		if IsEmptyDir(path) {
			_ = os.Remove(path)
			log.Printf("Removed empty directory: %s\n", path)
			return nil
		}

		return nil
	})
}

// IsEmptyDir checks if a directory is empty.
func IsEmptyDir(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) == 0
}

// IsJsonType checks if the content is a valid JSON document.
func IsJsonType(content []byte) bool {
	var jsonData map[string]interface{}
	return json.Unmarshal(content, &jsonData) == nil
}

// IsYamlType checks if the content is a valid YAML document.
func IsYamlType(content []byte) bool {
	var yamlData map[string]interface{}
	err := yaml.Unmarshal(content, &yamlData)
	return err == nil
}

// ExtractZip extracts a zip archive to a target directory.
// onlyPrefixes is a list of prefixes that are allowed to be extracted.
func ExtractZip(zipReader *zip.Reader, targetDir string, onlyPrefixes []string) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(zipReader.File))

	for _, zipFile := range zipReader.File {
		wg.Add(1)

		go func(zipFile *zip.File) {
			defer wg.Done()

			filePath := zipFile.Name
			takeIt := false

			for _, prefix := range onlyPrefixes {
				if strings.HasPrefix(filePath, prefix) {
					takeIt = true
					break
				}
			}

			if !takeIt {
				log.Printf("Skipping extracted file %s because it doesn't have allowed prefix\n", filePath)
				return
			}

			// Ensure the directory exists
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(filepath.Join(targetDir, dir), 0755); err != nil {
				errCh <- err
				return
			}

			targetPath := filepath.Join(targetDir, filePath)

			// Create the parent directory if it doesn't exist
			parentDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				errCh <- err
				return
			}

			// Extract and copy the file
			source, err := zipFile.Open()
			if err != nil {
				errCh <- err
				return
			}
			defer source.Close()

			target, err := os.Create(targetPath)
			if err != nil {
				errCh <- err
				return
			}
			defer target.Close()

			_, err = io.Copy(target, source)
			if err != nil {
				errCh <- err
				return
			}
		}(zipFile)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Collect errors
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

// GetFileHash gets the SHA256 hash of a file.
func GetFileHash(file io.Reader) string {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return fmt.Sprintf("%x", hash.Sum(nil))
}

func GetFileContentsFromURL(client *http.Client, url string) ([]byte, string, error) {
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", ErrGettingFileFromURL
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return content, contentType, nil
}
