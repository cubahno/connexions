package connexions

import (
	"fmt"
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
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		return true
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

func RemovePointer[T bool | float64 | int64 | uint64](value *T) T {
	var res T
	if value == nil {
		return res
	}
	return *value
}
