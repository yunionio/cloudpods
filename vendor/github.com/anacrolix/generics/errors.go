package generics

func UnwrapErrorTuple[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}
