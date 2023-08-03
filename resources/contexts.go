package resources

import _ "embed"

//go:embed contexts/common.yml
var CommonContextYAMLContents []byte

//go:embed contexts/fake.yml
var FakeContextYAMLContents []byte

//go:embed contexts/words.yml
var WordsContextYAMLContents []byte
