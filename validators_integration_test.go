//go:build integration

package connexions

import (
    assert2 "github.com/stretchr/testify/assert"
    "testing"
)

func TestValidateResponse_e2e(t *testing.T) {
    assert := assert2.New(t)

    assert.True(true)
}
