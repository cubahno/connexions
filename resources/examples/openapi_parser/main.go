package main

import (
	"fmt"
	"github.com/cubahno/connexions"
)

func main() {
	docFactory := connexions.NewDocumentFromFileFactory(connexions.LibOpenAPIProvider)
	doc, _ := docFactory("resources/petstore.yaml")
	fmt.Printf("Loaded document version %s, with %d resources\n", doc.GetVersion(), len(doc.GetResources()))
}
