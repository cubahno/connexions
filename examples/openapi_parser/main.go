package main

import (
	"fmt"

	"github.com/cubahno/connexions/openapi/provider"
)

func main() {
	doc, _ := provider.NewDocumentFromFile("resources/petstore.yaml")
	fmt.Printf("Loaded document version %s, with %d resources\n", doc.GetVersion(), len(doc.GetResources()))
}
