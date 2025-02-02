package main

import (
	"github.com/cubahno/connexions/internal"
)

func main() {
	// will create complete path if it does not exist
	_ = internal.SaveFile("/path/a/b/c/test.txt", []byte("hello world"))
	_ = internal.CopyFile("/path/a/b/c/test.txt", "/path/a/b/c/test2.txt")
	_ = internal.CopyDirectory("/path/a/b/c", "/path/a/b/c2")
}
