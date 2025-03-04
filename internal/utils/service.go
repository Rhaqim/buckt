package utils

func InterfaceSlice[T any](slice []T) []any {
	var s []any
	for _, v := range slice {
		s = append(s, v)
	}
	return s
}
