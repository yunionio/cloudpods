package generics

import (
	"golang.org/x/exp/constraints"
)

// Pops the last element from the slice and returns it. Panics if the slice is empty, or if the
// slice is nil.
func SlicePop[T any](slice *[]T) T {
	lastIndex := len(*slice) - 1
	last := (*slice)[lastIndex]
	*slice = (*slice)[:lastIndex]
	return last
}

func MakeSliceWithLength[T any, L constraints.Integer](slice *[]T, length L) {
	*slice = make([]T, length)
}

func MakeSliceWithCap[T any, L constraints.Integer](slice *[]T, cap L) {
	*slice = make([]T, 0, cap)
}

func Reversed[T any](slice []T) []T {
	reversed := make([]T, len(slice))
	for i := range reversed {
		reversed[i] = slice[len(slice)-1-i]
	}
	return reversed
}

func Singleton[T any](t T) []T {
	return []T{t}
}

// I take it there's no way to do this with a generic return slice element type.
func ConvertToSliceOfAny[T any](ts []T) (ret []any) {
	ret = make([]any, 0, len(ts))
	for _, t := range ts {
		ret = append(ret, t)
	}
	return
}

func ReverseSlice[T any](slice []T) {
	for i := 0; i < len(slice)/2; i++ {
		slice[i], slice[len(slice)-1-i] = slice[len(slice)-1-i], slice[i]
	}
}

func SliceTake[T any](n int, slice []T) []T {
	return slice[:Min(n, len(slice))]
}

func SliceDrop[T any](n int, slice []T) []T {
	return slice[Min(n, len(slice)):]
}

func SliceGet[T any, I constraints.Integer](slice []T, index I) (ret Option[T]) {
	if int(index) < len(slice) {
		ret = Some(slice[index])
	}
	return
}

// Surely you should just pass iterator functions around instead. Go sux.
func SliceMap[From, To any](froms []From, convert func(From) To) []To {
	tos := make([]To, 0, len(froms))
	for _, from := range froms {
		tos = append(tos, convert(from))
	}
	return tos
}
