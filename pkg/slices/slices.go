package sliceutils

func Map[T any, U any](values []T, mapper func(v T) U) []U {
	mapped := make([]U, len(values))
	for i, value := range values {
		mapped[i] = mapper(value)
	}
	return mapped
}
