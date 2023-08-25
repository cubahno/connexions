package connexions

import (
    "github.com/pb33f/libopenapi"
    v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
    "os"
)

func NewLibDocument(filePath string) (libopenapi.Document, error) {
    src, _ := os.ReadFile(filePath)

    // create a new document from specification bytes
    return libopenapi.NewDocument(src)
}

func NewLibModel(doc libopenapi.Document) (*libopenapi.DocumentModel[v3high.Document], []error) {
    return doc.BuildV3Model()
}
