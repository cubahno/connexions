package xs

// SliceDeleteAtIndex deletes an element from a slice at the given index and preserves the order of the slice.
func SliceDeleteAtIndex[T any](slice []T, index int) []T {
	return append(slice[:index], slice[index+1:]...)
}
