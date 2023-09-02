package connexions

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

func AppendSliceFirstNonEmpty[T comparable](data []T, value ...T) []T {
	var empty T

	for _, v := range value {
		if v != empty {
			return append(data, v)
		}
	}
	return data
}
