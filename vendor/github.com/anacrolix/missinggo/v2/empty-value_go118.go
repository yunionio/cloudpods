package missinggo

func IsZeroValue[T comparable](i T) bool {
	var z T
	return i == z
}
