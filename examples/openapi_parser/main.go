package main

import (
	"fmt"
	"github.com/cubahno/connexions"
	"github.com/cubahno/connexions/config"
)

func main() {
	docFactory := connexions.NewDocumentFromFileFactory(config.LibOpenAPIProvider)
	doc, _ := docFactory("resources/petstore.yaml")
	fmt.Printf("Loaded document version %s, with %d resources\n", doc.GetVersion(), len(doc.GetResources()))
}
