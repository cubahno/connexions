package main

import (
	"fmt"

	"github.com/cubahno/connexions/internal"
)

func main() {
	doc, _ := internal.NewDocumentFromFile("resources/petstore.yaml")
	fmt.Printf("Loaded document version %s, with %d resources\n", doc.GetVersion(), len(doc.GetResources()))
}
