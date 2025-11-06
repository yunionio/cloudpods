package generics

import "golang.org/x/exp/constraints"

// Deprecated: Use MakeMapIfNil and MapInsert separately.
func MakeMapIfNilAndSet[K comparable, V any](pm *map[K]V, k K, v V) (added bool) {
	MakeMapIfNil(pm)
	m := *pm
	_, exists := m[k]
	added = !exists
	m[k] = v
	return
}

// Does this exist in the maps package?
func MakeMap[K comparable, V any, M ~map[K]V](pm *M) {
	*pm = make(M)
}

func MakeMapWithCap[K comparable, V any, M ~map[K]V, C constraints.Integer](pm *M, cap C) {
	*pm = make(M, cap)
}

func MakeMapIfNil[K comparable, V any, M ~map[K]V](pm *M) {
	if *pm == nil {
		MakeMap(pm)
	}
}

func MapContains[K comparable, V any, M ~map[K]V](m M, k K) bool {
	_, ok := m[k]
	return ok
}

func MapMustGet[K comparable, V any, M ~map[K]V](m M, k K) V {
	v, ok := m[k]
	if !ok {
		panic(k)
	}
	return v
}

func MapInsert[K comparable, V any, M ~map[K]V](m M, k K, v V) Option[V] {
	old, ok := m[k]
	m[k] = v
	return Option[V]{
		Value: old,
		Ok:    ok,
	}
}
