package dynamic

// Multiply creates a slice of pointers to copies of the provided value.
// It takes a value of type T and a count, returning a slice with that many
// pointers to copies of the original value.
// Example: multiply(dockerImage, 100) returns a slice with 100 elements,
// each being a pointer to a copy of dockerImage.
func Multiply[T any](value T, count int) []T {
	if count <= 0 {
		return []T{}
	}

	result := make([]T, count)
	for i := range result {
		result[i] = value
	}
	return result
}
