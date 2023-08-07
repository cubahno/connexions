package xs

import (
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
