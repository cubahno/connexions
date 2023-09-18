package connexions

import (
	"fmt"
	"math"
	"reflect"
)

func IsNumber(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

func IsInteger(value interface{}) bool {
	switch v := value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return true
	case float32:
		return v == float32(math.Trunc(float64(v)))
	case float64:
		return value == math.Trunc(v)
	default:

		return false
	}
}

func ToFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int, int8, int16, int32, int64:
		return float64(reflect.ValueOf(v).Int()), nil
	case uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(v).Uint()), nil
	case float32, float64:
		return reflect.ValueOf(v).Float(), nil
	default:
		return 0, fmt.Errorf("unsupported type: %s", reflect.TypeOf(value))
	}
}

// ToInt32 converts underlying value to int32 if it can be represented as int32
func ToInt32(value any) (int32, bool) {
	switch v := value.(type) {
	case uint:
		if v <= math.MaxInt32 {
			return int32(v), true
		}
		return 0, false
	case uint8:
		return int32(v), true
	case uint16:
		return int32(v), true
	case uint32:
		if v <= math.MaxInt32 {
			return int32(v), true
		}
		return 0, false
	case uint64:
		if v <= math.MaxInt32 {
			return int32(v), true
		}
	case int:
		if v >= math.MinInt32 && v <= math.MaxInt32 {
			return int32(v), true
		}
	case int8:
		return int32(v), true
	case int16:
		return int32(v), true
	case int32:
		return v, true
	case int64:
		if v >= math.MinInt32 && v <= math.MaxInt32 {
			return int32(v), true
		}
	case float32:
		intValue := int32(v)
		if float32(intValue) == v {
			return intValue, true
		}
	case float64:
		intValue := int32(v)
		if float64(intValue) == v {
			return intValue, true
		}
	}
	return 0, false
}

// ToInt64 converts underlying value to int64 if it can be represented as int64
func ToInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float32:
		intValue := int64(v)
		if float32(intValue) == v {
			return intValue, true
		}
	case float64:
		intValue := int64(v)
		if float64(intValue) == v {
			return intValue, true
		}
	}
	return 0, false
}

func RemovePointer[T bool | float64 | int64 | uint64](value *T) T {
	var res T
	if value == nil {
		return res
	}
	return *value
}
