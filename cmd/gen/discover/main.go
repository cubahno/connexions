package main

import (
	"flag"
	"fmt"
	"os"

	cmdapi "github.com/cubahno/connexions/v2/cmd/api"
)

func main() {
	flag.Parse()

	err := cmdapi.Discover(cmdapi.DiscoverOptions{})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
