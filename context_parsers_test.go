//go:build !integration

package connexions

import (
	assert2 "github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestParseContextFile(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("happy-path", func(t *testing.T) {
		tempDir := t.TempDir()
		contents := `
name: Jane
age: 30
job: fake:company.job_title
shift: fake:shift
nickname: alias:fake.gamer.tag
motto: "botify:?????? ###!"
tv-show: func:echo:Sanford & Son
`
		filePath := filepath.Join(tempDir, "params.yml")
		err := os.WriteFile(filePath, []byte(contents), 0644)
		assert.Nil(err)

		res, err := ParseContextFile(filePath, Fakes)
		assert.Nil(err)

		results := res.Result
		aliases := res.Aliases

		assert.Equal("Jane", results["name"])
		assert.Equal(30, results["age"])

		jobFn, ok := results["job"].(FakeFunc)
		assert.True(ok)
		job := jobFn().Get().(string)
		assert.Greater(len(job), 0)

		shiftFn, ok := results["shift"].(FakeFunc)
		assert.False(ok)
		assert.Nil(shiftFn)
		shiftVal, ok := results["shift"].(string)
		assert.True(ok)
		assert.Equal("fake:shift", shiftVal)

		// it's still there, not replaced yet
		nickname, ok := results["nickname"].(string)
		assert.True(ok)
		assert.Equal("alias:fake.gamer.tag", nickname)

		// aliases resolved in different place.
		// here they are just collected
		assert.Equal(map[string]string{
			"nickname": "fake.gamer.tag",
		}, aliases)

		mottoFn, ok := results["motto"].(FakeFunc)
		assert.True(ok)
		motto := mottoFn().Get().(string)
		assert.Equal(len(motto), 11)

		tvShowFn, ok := results["tv-show"].(FakeFunc)
		assert.True(ok)
		tvShow := tvShowFn().Get().(string)
		assert.Equal("Sanford & Son", tvShow)
	})

	t.Run("bad-file", func(t *testing.T) {
		res, err := ParseContextFile("bad-file.yml", Fakes)
		assert.Nil(res)
		assert.NotNil(err)
	})
}

func TestParseContexFromBytes(t *testing.T) {
	assert := assert2.New(t)

	t.Run("happy-path", func(t *testing.T) {
		contents := `
name: Jane
job: fake:company.job_title
hallo: func:echo:Welt!
`
		res, err := ParseContextFromBytes([]byte(contents), Fakes)
		assert.Nil(err)

		results := res.Result
		aliases := res.Aliases

		assert.Equal(map[string]string{}, aliases)

		assert.Equal("Jane", results["name"])

		jobFn, ok := results["job"].(FakeFunc)
		assert.True(ok)
		job := jobFn().Get().(string)
		assert.Greater(len(job), 0)

		echoFn, ok := results["hallo"].(FakeFunc)
		assert.True(ok)
		echo := echoFn().Get().(string)
		assert.Equal("Welt!", echo)
	})

	t.Run("invalid-yaml", func(t *testing.T) {
		res, err := ParseContextFromBytes([]byte(`1`), Fakes)
		assert.Nil(res)
		assert.NotNil(err)
	})
}

func TestCollectContexts(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	names := []map[string]string{
		{"common": ""},
		{"service": "person"},
	}

	fileCollections := map[string]map[string]any{
		"common": {
			"name": "Jane",
			"age":  30,
			"company": map[string]any{
				"name": "func:make_company_name",
			},
		},
		"service": {
			"person": map[string]any{
				"job":  "fake:company.job_title",
				"name": "John",
				"age":  40,
			},
			"company": map[string]any{
				"name": "fake:company.name",
			},
		},
		"other": {
			"id": "fake:uuid",
		},
	}

	initial := map[string]any{
		"city": "New York",
	}

	expected := []map[string]any{
		// initial has top precedence
		{
			"city": "New York",
		},
		// complete common context
		{
			"name": "Jane",
			"age":  30,
			"company": map[string]any{
				"name": "func:make_company_name",
			},
		},
		// service.person only
		{
			"job":  "fake:company.job_title",
			"name": "John",
			"age":  40,
		},
	}

	res := CollectContexts(names, fileCollections, initial)

	assert.Equal(expected, res)
}

func TestParseFakeContextFunc(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("insufficient-parts", func(t *testing.T) {
		res, ok := parseFakeContextFunc("key", []string{"fake"}, nil)
		assert.Nil(res)
		assert.False(ok)
	})

	t.Run("empty-available", func(t *testing.T) {
		res, ok := parseFakeContextFunc("some.id2", []string{"fake", "some.id"}, nil)
		assert.Nil(res)
		assert.False(ok)
	})

	t.Run("is-available", func(t *testing.T) {
		available := map[string]FakeFunc{
			"some.id": func() MixedValue {
				return IntValue(1212)
			},
		}
		res, ok := parseFakeContextFunc("some.id", []string{"fake", "some.id"}, available)
		assert.NotNil(res)
		assert.True(ok)
		assert.Equal(int64(1212), res().Get())
	})

	t.Run("is-available-with-empty-name", func(t *testing.T) {
		available := map[string]FakeFunc{
			"some.id": func() MixedValue {
				return IntValue(1212)
			},
		}
		res, ok := parseFakeContextFunc("some.id", []string{"fake", ""}, available)
		assert.NotNil(res)
		assert.True(ok)
		assert.Equal(int64(1212), res().Get())
	})

	t.Run("not-available", func(t *testing.T) {
		available := map[string]FakeFunc{
			"some.id": func() MixedValue {
				return IntValue(1212)
			},
		}
		res, ok := parseFakeContextFunc("id2", []string{"fake", "some.id2"}, available)
		assert.Nil(res)
		assert.False(ok)
	})
}

func TestParseOneArgContextFunc(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("insufficient-parts", func(t *testing.T) {
		res, ok := parseOneArgContextFunc([]string{"fake"}, nil)
		assert.Nil(res)
		assert.False(ok)
	})

	t.Run("is-available", func(t *testing.T) {
		available := map[string]FakeFuncFactoryWithString{
			"hello": func(value string) FakeFunc {
				return func() MixedValue {
					return StringValue("Hello, " + value + "!")
				}
			},
		}
		res, ok := parseOneArgContextFunc([]string{"func", "hello", "Motto"}, available)
		assert.NotNil(res)
		assert.True(ok)
		assert.Equal("Hello, Motto!", res().Get())
	})

	t.Run("not-available", func(t *testing.T) {
		available := map[string]FakeFuncFactoryWithString{
			"hello": func(value string) FakeFunc {
				return func() MixedValue {
					return StringValue("Hello, " + value + "!")
				}
			},
		}
		res, ok := parseOneArgContextFunc([]string{"func", "hello2", "Motto"}, available)
		assert.Nil(res)
		assert.False(ok)
	})
}

func TestParseBotifyContextFunc(t *testing.T) {
	assert := assert2.New(t)
	t.Parallel()

	t.Run("insufficient-parts", func(t *testing.T) {
		res, ok := parseBotifyContextFunc([]string{"botify"}, nil)
		assert.Nil(res)
		assert.False(ok)
	})

	t.Run("empty-pattern", func(t *testing.T) {
		res, ok := parseBotifyContextFunc([]string{"botify", ""}, nil)
		assert.Nil(res)
		assert.False(ok)
	})

	t.Run("is-available", func(t *testing.T) {
		available := map[string]FakeFuncFactoryWithString{
			"botify": func(value string) FakeFunc {
				return func() MixedValue {
					return StringValue("botified")
				}
			},
		}
		res, ok := parseBotifyContextFunc([]string{"botify", "???"}, available)
		assert.NotNil(res)
		assert.True(ok)
		assert.Equal("botified", res().Get())
	})
}
