package main

import "github.com/cubahno/connexions"

func main() {
	// will create complete path if it does not exist
	_ = connexions.SaveFile("/path/a/b/c/test.txt", []byte("hello world"))
	_ = connexions.CopyFile("/path/a/b/c/test.txt", "/path/a/b/c/test2.txt")
	_ = connexions.CopyDirectory("/path/a/b/c", "/path/a/b/c2")
}
