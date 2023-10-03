package main

import (
	"fmt"
	"github.com/cubahno/connexions/contexts"
)

func main() {
	fakeMap := contexts.GetFakes()
	uuid := fakeMap["uuid.v4"]().Get()
	tag := fakeMap["gamer.tag"]().Get()
	fmt.Printf("uuid: %s, tag: %v\n", uuid, tag)
}
