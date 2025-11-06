package heap

type sliceInterface[T any] struct {
	slice *[]T
	less  func(T, T) bool
}

// This is for use by the heap package's global functions, you probably don't mean to call this
// directly.
func (me sliceInterface[T]) Less(i, j int) bool {
	return me.less((*me.slice)[i], (*me.slice)[j])
}

// This is for use by the heap package's global functions, you probably don't mean to call this
// directly.
func (me sliceInterface[T]) Swap(i, j int) {
	s := *me.slice
	s[i], s[j] = s[j], s[i]
	*me.slice = s
}

// This is for use by the heap package's global functions, you probably don't mean to call this
// directly.
func (me sliceInterface[T]) Push(x T) {
	*me.slice = append(*me.slice, x)
}

// This is for use by the heap package's global functions, you probably don't mean to call this
// directly.
func (me sliceInterface[T]) Pop() T {
	s := *me.slice
	n := len(s)
	ret := s[n-1]
	*me.slice = s[:n-1]
	return ret
}

func (me sliceInterface[T]) Len() int {
	return len(*me.slice)
}

// Creates an Interface that operates on a slice in place. The Interface should be used with the
// heap package's global functions just like you would with a manual implementation of Interface for
// a slice. i.e. don't call Interface.{Push,Pop}, call heap.{Push,Pop} and pass the return value
// from this function.
func InterfaceForSlice[T any](sl *[]T, less func(l T, r T) bool) Interface[T] {
	return sliceInterface[T]{
		slice: sl,
		less:  less,
	}
}
