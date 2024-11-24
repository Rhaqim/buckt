package utils

func InterfaceSlice(slice interface{}) []interface{} {
	s := slice.([]interface{})
	return s
}
