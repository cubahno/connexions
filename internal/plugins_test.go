package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompilePlugin(t *testing.T) {
	dir := t.TempDir()
	src := `
package main
func Ping() string {
	return "Pong"
}
`
	filePath := filepath.Join(dir, "foo.go")
	_ = os.WriteFile(filePath, []byte(src), 0644)

	p, err := CompilePlugin(dir)
	if err != nil {
		t.Fatal(err)
		return
	}

	symbol, err := p.Lookup("Ping")
	if err != nil {
		t.Fatal(err)
		return
	}

	fn, ok := symbol.(func() string)
	assert.True(t, ok)
	assert.Equal(t, "Pong", fn())

	symbol, err = p.Lookup("NonExiting")
	assert.Nil(t, symbol)
	assert.NotNil(t, err)
}
