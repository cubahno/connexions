//go:build !integration

package connexions

import (
	"fmt"
	assert2 "github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestIsNumber(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected bool
	}{
		{42, true},
		{int8(127), true},
		{int16(-1000), true},
		{int32(20000), true},
		{int64(123456789), true},
		{uint(123), true},
		{uint8(255), true},
		{uint16(65535), true},
		{uint32(4294967295), true},
		{uint64(18446744073709551615), true},
		{float32(3.14), true},
		{float64(math.Pi), true},
		{"hello", false},
		{true, false},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			actual := IsNumber(tc.input)
			if actual != tc.expected {
				t.Errorf("IsNumber(%v) - Expected: %v, Got: %v", tc.input, tc.expected, actual)
			}
		})
	}
}

func TestIsInteger(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected bool
	}{
		{42, true},
		{int8(127), true},
		{int16(-1000), true},
		{int32(20000), true},
		{int64(123456789), true},
		{uint(123), true},
		{uint8(255), true},
		{uint16(65535), true},
		{uint32(4294967295), true},
		{uint64(18446744073709551615), true},
		{float32(3), true},
		{float64(3), true},
		{float32(3.14), false},
		{float64(math.Pi), false},
		{"hello", false},
		{true, false},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			actual := IsInteger(tc.input)
			if actual != tc.expected {
				t.Errorf("IsInteger(%v) - Expected: %v, Got: %v", tc.input, tc.expected, actual)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	assert := assert2.New(t)

	tests := []struct {
		input    interface{}
		expected float64
	}{
		{int32(42), 42.0},
		{int64(123456), 123456.0},
		{uint8(255), 255.0},
		{float32(3.14), 3.14},
		{float64(2.718), 2.718},
	}

	for _, test := range tests {
		result, err := ToFloat64(test.input)
		if err != nil {
			t.Errorf("Error converting %v: %v", test.input, err)
			continue
		}

		expectedStr := fmt.Sprintf("%.6f", test.expected)
		resultStr := fmt.Sprintf("%.6f", result)

		if resultStr != expectedStr {
			t.Errorf("For input %v, expected %s but got %s", test.input, expectedStr, resultStr)
		}
	}

	val, err := ToFloat64("string")
	assert.Equal(0.0, val)
	assert.NotNil(err)
}

func TestToInt32(t *testing.T) {
	assert := assert2.New(t)

	okTests := []struct {
		input    interface{}
		expected int32
	}{
		{30, 30},
		{int8(21), 21},
		{int16(42), 42},
		{int32(42), 42},
		{int64(123456), 123456},
		{uint(12), 12},
		{uint8(255), 255},
		{uint16(255), 255},
		{uint32(255), 255},
		{uint64(255), 255},
		{int64(123456), 123456},
		{float32(3.0), 3},
		{float64(3.0), 3},
	}

	for _, test := range okTests {
		result, ok := ToInt32(test.input)
		assert.True(ok)
		assert.Equal(test.expected, result)
	}

	notOkTests := []struct {
		input    interface{}
	}{
		{"30"},
		{"hello"},
		{true},
	}

	for _, test := range notOkTests {
		result, ok := ToInt32(test.input)
		assert.False(ok)
		assert.Equal(int32(0), result)
	}
}

func TestToInt64(t *testing.T) {
	assert := assert2.New(t)

	okTests := []struct {
		input    interface{}
		expected int64
	}{
		{30, 30},
		{int8(21), 21},
		{int16(31), 31},
		{int32(32), 32},
		{int64(33), 33},
		{uint(21), 21},
		{uint8(34), 34},
		{uint16(35), 35},
		{uint32(36), 36},
		{uint64(37), 37},
		{float32(3.0), 3},
		{float64(3.0), 3},
	}

	for _, test := range okTests {
		result, ok := ToInt64(test.input)
		if !ok {
			t.Errorf("Expected %v to be ok", test.input)
			t.Fail()
		}
		assert.Equal(test.expected, result)
	}

	notOkTests := []struct {
		input    interface{}
	}{
		{"30"},
		{"hello"},
		{true},
	}

	for _, test := range notOkTests {
		result, ok := ToInt64(test.input)
		assert.False(ok)
		assert.Equal(int64(0), result)
	}
}

func TestRemovePointer(t *testing.T) {
	assert := assert2.New(t)

	t.Run("nil-bool", func(t *testing.T) {
		res := RemovePointer[bool](nil)
		assert.Equal(false, res)
	})

	t.Run("nil-float64", func(t *testing.T) {
		res := RemovePointer[float64](nil)
		assert.Equal(0.0, res)
	})

	t.Run("int64", func(t *testing.T) {
		v := int64(21)
		ptr := &v
		res := RemovePointer(ptr)
		assert.Equal(v, res)
	})
}
