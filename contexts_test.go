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

    res, err := ParseContextFile(filePath)
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
    shiftVal, ok :=  results["shift"].(string)
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
}

func TestParseContexFromBytes(t *testing.T) {
    assert := assert2.New(t)
    t.Parallel()

    contents := `
name: Jane
job: fake:company.job_title
hallo: func:echo:Welt!
`
    res, err := ParseContextFromBytes([]byte(contents))
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
}

func TestCollectContexts(t *testing.T) {
    assert := assert2.New(t)
    t.Parallel()

    assert.True(true)
}
