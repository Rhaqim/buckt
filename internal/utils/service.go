package utils

func InterfaceSlice[T any](slice []T) []interface{} {
	var s []interface{}
	for _, v := range slice {
		s = append(s, v)
	}
	return s
}
