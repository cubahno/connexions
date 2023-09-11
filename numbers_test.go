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
