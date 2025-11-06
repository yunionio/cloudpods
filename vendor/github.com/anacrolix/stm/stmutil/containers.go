package stmutil

import (
	"unsafe"

	"github.com/benbjohnson/immutable"

	"github.com/anacrolix/missinggo/v2/iter"
)

type Settish interface {
	Add(any) Settish
	Delete(any) Settish
	Contains(any) bool
	Range(func(any) bool)
	iter.Iterable
	Len() int
}

type mapToSet struct {
	m Mappish
}

type interhash struct{}

func (interhash) Hash(x any) uint32 {
	return uint32(nilinterhash(unsafe.Pointer(&x), 0))
}

func (interhash) Equal(i, j any) bool {
	return i == j
}

func NewSet() Settish {
	return mapToSet{NewMap()}
}

func NewSortedSet(lesser lessFunc) Settish {
	return mapToSet{NewSortedMap(lesser)}
}

func (s mapToSet) Add(x any) Settish {
	s.m = s.m.Set(x, nil)
	return s
}

func (s mapToSet) Delete(x any) Settish {
	s.m = s.m.Delete(x)
	return s
}

func (s mapToSet) Len() int {
	return s.m.Len()
}

func (s mapToSet) Contains(x any) bool {
	_, ok := s.m.Get(x)
	return ok
}

func (s mapToSet) Range(f func(any) bool) {
	s.m.Range(func(k, _ any) bool {
		return f(k)
	})
}

func (s mapToSet) Iter(cb iter.Callback) {
	s.Range(cb)
}

type Map struct {
	*immutable.Map
}

func NewMap() Mappish {
	return Map{immutable.NewMap(interhash{})}
}

var _ Mappish = Map{}

func (m Map) Delete(x any) Mappish {
	m.Map = m.Map.Delete(x)
	return m
}

func (m Map) Set(key, value any) Mappish {
	m.Map = m.Map.Set(key, value)
	return m
}

func (sm Map) Range(f func(key, value any) bool) {
	iter := sm.Map.Iterator()
	for !iter.Done() {
		if !f(iter.Next()) {
			return
		}
	}
}

func (sm Map) Iter(cb iter.Callback) {
	sm.Range(func(key, _ any) bool {
		return cb(key)
	})
}

type SortedMap struct {
	*immutable.SortedMap
}

func (sm SortedMap) Set(key, value any) Mappish {
	sm.SortedMap = sm.SortedMap.Set(key, value)
	return sm
}

func (sm SortedMap) Delete(key any) Mappish {
	sm.SortedMap = sm.SortedMap.Delete(key)
	return sm
}

func (sm SortedMap) Range(f func(key, value any) bool) {
	iter := sm.SortedMap.Iterator()
	for !iter.Done() {
		if !f(iter.Next()) {
			return
		}
	}
}

func (sm SortedMap) Iter(cb iter.Callback) {
	sm.Range(func(key, _ any) bool {
		return cb(key)
	})
}

type lessFunc func(l, r any) bool

type comparer struct {
	less lessFunc
}

func (me comparer) Compare(i, j any) int {
	if me.less(i, j) {
		return -1
	} else if me.less(j, i) {
		return 1
	} else {
		return 0
	}
}

func NewSortedMap(less lessFunc) Mappish {
	return SortedMap{
		SortedMap: immutable.NewSortedMap(comparer{less}),
	}
}

type Mappish interface {
	Set(key, value any) Mappish
	Delete(key any) Mappish
	Get(key any) (any, bool)
	Range(func(_, _ any) bool)
	Len() int
	iter.Iterable
}

func GetLeft(l, _ any) any {
	return l
}

//go:noescape
//go:linkname nilinterhash runtime.nilinterhash
func nilinterhash(p unsafe.Pointer, h uintptr) uintptr

func interfaceHash(x any) uint32 {
	return uint32(nilinterhash(unsafe.Pointer(&x), 0))
}

type Lenner interface {
	Len() int
}

type List = *immutable.List
