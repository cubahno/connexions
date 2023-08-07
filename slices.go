package xs

func IsSlicesEqual[T comparable](slice1, slice2 []T) bool {
    if len(slice1) != len(slice2) {
        return false
    }
    for i, value := range slice1 {
        if value != slice2[i] {
            return false
        }
    }
    return true
}
