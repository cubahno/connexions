package types

import (
	"math/rand"
)

// SliceDeleteAtIndex deletes an element from a slice at the given index and preserves the order of the slice.
func SliceDeleteAtIndex[T any](slice []T, index int) []T {
	if index < 0 || index >= len(slice) {
		return slice
	}
	return append(slice[:index], slice[index+1:]...)
}

// GetRandomSliceValue returns a random value from the given slice.
func GetRandomSliceValue[T any](slice []T) T {
	var res T
	if len(slice) == 0 {
		return res
	}
	return slice[rand.Intn(len(slice))]
}

// SliceContains returns true if the given slice contains the given value.
func SliceContains[T comparable](slice []T, value T) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// SliceUnique returns a new slice with unique values from the given slice.
func SliceUnique[T comparable](slice []T) []T {
	visited := make(map[T]bool)
	var result []T
	for _, item := range slice {
		if _, ok := visited[item]; !ok {
			visited[item] = true
			result = append(result, item)
		}
	}
	return result
}

// IsSliceUnique returns true if all values in the given slice are unique.
func IsSliceUnique[T comparable](path []T) bool {
	visited := make(map[T]bool)
	for _, item := range path {
		if _, ok := visited[item]; ok {
			return false
		}
		visited[item] = true
	}
	return true
}

// GetSliceMaxRepetitionNumber returns the maximum number of non-unique values in the given slice.
func GetSliceMaxRepetitionNumber[T comparable](values []T) int {
	max := 0

	if len(values) == 0 || len(values) == 1 {
		return max
	}

	visited := make(map[T]int)
	for _, item := range values {
		visited[item]++
	}

	for _, value := range visited {
		if value > max {
			max = value
		}
	}

	if max > 0 {
		max--
	}

	return max
}

// AppendSliceFirstNonEmpty appends the first non-empty value to the given slice.
func AppendSliceFirstNonEmpty[T comparable](data []T, value ...T) []T {
	var empty T

	for _, v := range value {
		if v != empty {
			return append(data, v)
		}
	}
	return data
}
