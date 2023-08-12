package xs

func SliceDeleteAtIndex[T any](slice []T, index int) []T {
	return append(slice[:index], slice[index+1:]...)
}
