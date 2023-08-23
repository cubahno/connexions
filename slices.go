package connexions

import (
	"math/rand"
)

// SliceDeleteAtIndex deletes an element from a slice at the given index and preserves the order of the slice.
func SliceDeleteAtIndex[T any](slice []T, index int) []T {
	return append(slice[:index], slice[index+1:]...)
}

func GetRandomSliceValue[T any](slice []T) T {
	var res T
	if len(slice) == 0 {
		return res
	}
	return slice[rand.Intn(len(slice))]
}

func SliceContains[T comparable](slice []T, value T) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
