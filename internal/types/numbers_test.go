//go:build !integration

package types

import (
	"fmt"
	"math"
	"testing"

	assert2 "github.com/stretchr/testify/assert"
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
		input interface{}
	}{
		{"30"},
		{"hello"},
		{true},
		{^uint(0) >> 1},
		{^uint32(0)>>1 + 1},
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
		input interface{}
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

func TestToUint8(t *testing.T) {
	assert := assert2.New(t)

	okTests := []struct {
		input    interface{}
		expected uint8
	}{
		{uint8(255), 255},
		{uint16(200), 200},
		{uint32(100), 100},
		{uint64(50), 50},
		{uint(25), 25},
		{int(100), 100},
		{int8(100), 100},
		{int16(200), 200},
		{int32(255), 255},
		{int64(128), 128},
		{float32(100.0), 100},
		{float64(200.0), 200},
	}

	for _, test := range okTests {
		result, ok := ToUint8(test.input)
		assert.True(ok, "Expected %v to be ok", test.input)
		assert.Equal(test.expected, result)
	}

	notOkTests := []struct {
		input interface{}
	}{
		{"30"},
		{int(-1)},
		{int8(-1)},
		{uint16(256)},
		{uint32(1000)},
		{float64(256.0)},
		{float64(100.5)},
	}

	for _, test := range notOkTests {
		result, ok := ToUint8(test.input)
		assert.False(ok, "Expected %v to not be ok", test.input)
		assert.Equal(uint8(0), result)
	}
}

func TestToUint16(t *testing.T) {
	assert := assert2.New(t)

	okTests := []struct {
		input    interface{}
		expected uint16
	}{
		{uint8(255), 255},
		{uint16(65535), 65535},
		{uint32(1000), 1000},
		{uint64(500), 500},
		{uint(250), 250},
		{int(1000), 1000},
		{int8(100), 100},
		{int16(32000), 32000},
		{int32(65535), 65535},
		{int64(12800), 12800},
		{float32(1000.0), 1000},
		{float64(2000.0), 2000},
	}

	for _, test := range okTests {
		result, ok := ToUint16(test.input)
		assert.True(ok, "Expected %v to be ok", test.input)
		assert.Equal(test.expected, result)
	}

	notOkTests := []struct {
		input interface{}
	}{
		{"30"},
		{int(-1)},
		{int8(-1)},
		{uint32(65536)},
		{float64(65536.0)},
		{float64(100.5)},
	}

	for _, test := range notOkTests {
		result, ok := ToUint16(test.input)
		assert.False(ok, "Expected %v to not be ok", test.input)
		assert.Equal(uint16(0), result)
	}
}

func TestToUint32(t *testing.T) {
	assert := assert2.New(t)

	okTests := []struct {
		input    interface{}
		expected uint32
	}{
		{uint8(255), 255},
		{uint16(65535), 65535},
		{uint32(4294967295), 4294967295},
		{uint64(1000000), 1000000},
		{uint(250), 250},
		{int(1000000), 1000000},
		{int8(100), 100},
		{int16(32000), 32000},
		{int32(2147483647), 2147483647},
		{int64(1280000), 1280000},
		{float32(1000.0), 1000},
		{float64(2000000.0), 2000000},
	}

	for _, test := range okTests {
		result, ok := ToUint32(test.input)
		assert.True(ok, "Expected %v to be ok", test.input)
		assert.Equal(test.expected, result)
	}

	notOkTests := []struct {
		input interface{}
	}{
		{"30"},
		{int(-1)},
		{int8(-1)},
		{uint64(4294967296)},
		{float64(4294967296.0)},
		{float64(100.5)},
	}

	for _, test := range notOkTests {
		result, ok := ToUint32(test.input)
		assert.False(ok, "Expected %v to not be ok", test.input)
		assert.Equal(uint32(0), result)
	}
}

func TestToUint64(t *testing.T) {
	assert := assert2.New(t)

	okTests := []struct {
		input    interface{}
		expected uint64
	}{
		{uint8(255), 255},
		{uint16(65535), 65535},
		{uint32(4294967295), 4294967295},
		{uint64(18446744073709551615), 18446744073709551615},
		{uint(250), 250},
		{int(1000000), 1000000},
		{int8(100), 100},
		{int16(32000), 32000},
		{int32(2147483647), 2147483647},
		{int64(9223372036854775807), 9223372036854775807},
		{float32(1000.0), 1000},
		{float64(2000000.0), 2000000},
	}

	for _, test := range okTests {
		result, ok := ToUint64(test.input)
		assert.True(ok, "Expected %v to be ok", test.input)
		assert.Equal(test.expected, result)
	}

	notOkTests := []struct {
		input interface{}
	}{
		{"30"},
		{int(-1)},
		{int8(-1)},
		{float64(100.5)},
	}

	for _, test := range notOkTests {
		result, ok := ToUint64(test.input)
		assert.False(ok, "Expected %v to not be ok", test.input)
		assert.Equal(uint64(0), result)
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
