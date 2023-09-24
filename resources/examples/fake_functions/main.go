package main

import (
    "fmt"
    "github.com/cubahno/connexions"
)

func main() {
    fakeMap := connexions.GetFakes()
    uuid := fakeMap["uuid.v4"]().Get()
    tag := fakeMap["gamer.tag"]().Get()
    fmt.Printf("uuid: %s, tag: %v\n", uuid, tag)
}
