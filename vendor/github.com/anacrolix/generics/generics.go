package generics

import "golang.org/x/exp/constraints"

func InitNew[T any](p **T) {
	*p = new(T)
}

func SetZero[T any](p *T) {
	*p = ZeroValue[T]()
}

func PtrTo[T any](t T) *T {
	return &t
}

// Returns a zero-size, zero-allocation slice of the given length that can be used with range to
// loop n times. Also has the advantage of not requiring a loop variable. Similar to bradfitz's
// iter.N, and my clone in anacrolix/missinggo.
func Range[T constraints.Integer](n T) []struct{} {
	return make([]struct{}, n)
}
