// Package atomic contains type-safe atomic types.
//
// The zero value for the numeric types cannot be used. Use New*. The
// rationale for this behaviour is that copying an atomic integer is not
// reliable. Copying can be prevented by embedding sync.Mutex, but that bloats
// the type.
package atomic

import "sync/atomic"

// Interface represents atomic operations on a value.
type Interface[T any] interface {
	// Load value atomically.
	Load() T
	// Store value atomically.
	Store(value T)
	// Swap the previous value with the new value atomically.
	Swap(new T) (old T)
	// CompareAndSwap the previous value with new if its value is "old".
	CompareAndSwap(old, new T) (swapped bool)
}

var _ Interface[bool] = &Value[bool]{}

// Value wraps any generic value in atomic load and store operations.
type Value[T any] struct {
	value atomic.Value
}

// New atomic Value.
func New[T any](seed T) *Value[T] {
	v := &Value[T]{}
	v.value.Store(seed)
	return v
}

func (v *Value[T]) Load() (out T) {
	value := v.value.Load()
	if value == nil {
		return out
	}
	return value.(T)
}
func (v *Value[T]) Store(value T)                            { v.value.Store(value) }
func (v *Value[T]) Swap(new T) (old T)                       { return v.value.Swap(new).(T) }
func (v *Value[T]) CompareAndSwap(old, new T) (swapped bool) { return v.value.CompareAndSwap(old, new) }

// atomicint defines the types that atomic integer operations are supported on.
type atomicint interface {
	int32 | uint32 | int64 | uint64
}

// Int expresses atomic operations on signed or unsigned integer values.
type Int[T atomicint] interface {
	Interface[T]
	// Add a value and return the new result.
	Add(delta T) (new T)
}

// Currently not supported by Go's generic type system:
//
//    ./atomic.go:48:9: cannot use type switch on type parameter value v (variable of type T constrained by atomicint)
//
// // ForInt infers and creates an atomic Int[T] type for a value.
// func ForInt[T atomicint](v T) Int[T] {
// 	switch v.(type) {
// 	case int32:
// 		return NewInt32(v)
// 	case uint32:
// 		return NewUint32(v)
// 	case int64:
// 		return NewInt64(v)
// 	case uint64:
// 		return NewUint64(v)
// 	}
// 	panic("can't happen")
// }

// Int32 atomic value.
//
// Copying creates an alias. The zero value is not usable, use NewInt32.
type Int32 struct{ value *int32 }

// NewInt32 creates a new atomic integer with an initial value.
func NewInt32(value int32) Int32 { return Int32{value: &value} }

var _ Int[int32] = &Int32{}

func (i Int32) Add(delta int32) (new int32) { return atomic.AddInt32(i.value, delta) }
func (i Int32) Load() (val int32)           { return atomic.LoadInt32(i.value) }
func (i Int32) Store(val int32)             { atomic.StoreInt32(i.value, val) }
func (i Int32) Swap(new int32) (old int32)  { return atomic.SwapInt32(i.value, new) }
func (i Int32) CompareAndSwap(old, new int32) (swapped bool) {
	return atomic.CompareAndSwapInt32(i.value, old, new)
}

// Uint32 atomic value.
//
// Copying creates an alias.
type Uint32 struct{ value *uint32 }

var _ Int[uint32] = Uint32{}

// NewUint32 creates a new atomic integer with an initial value.
func NewUint32(value uint32) Uint32 { return Uint32{value: &value} }

func (i Uint32) Add(delta uint32) (new uint32) { return atomic.AddUint32(i.value, delta) }
func (i Uint32) Load() (val uint32)            { return atomic.LoadUint32(i.value) }
func (i Uint32) Store(val uint32)              { atomic.StoreUint32(i.value, val) }
func (i Uint32) Swap(new uint32) (old uint32)  { return atomic.SwapUint32(i.value, new) }
func (i Uint32) CompareAndSwap(old, new uint32) (swapped bool) {
	return atomic.CompareAndSwapUint32(i.value, old, new)
}

// Int64 atomic value.
//
// Copying creates an alias.
type Int64 struct{ value *int64 }

var _ Int[int64] = Int64{}

// NewInt64 creates a new atomic integer with an initial value.
func NewInt64(value int64) Int64 { return Int64{value: &value} }

func (i Int64) Add(delta int64) (new int64) { return atomic.AddInt64(i.value, delta) }
func (i Int64) Load() (val int64)           { return atomic.LoadInt64(i.value) }
func (i Int64) Store(val int64)             { atomic.StoreInt64(i.value, val) }
func (i Int64) Swap(new int64) (old int64)  { return atomic.SwapInt64(i.value, new) }
func (i Int64) CompareAndSwap(old, new int64) (swapped bool) {
	return atomic.CompareAndSwapInt64(i.value, old, new)
}

// Uint64 atomic value.
//
// Copying creates an alias.
type Uint64 struct{ value *uint64 }

var _ Int[uint64] = Uint64{}

// NewUint64 creates a new atomic integer with an initial value.
func NewUint64(value uint64) Uint64 { return Uint64{value: &value} }

func (i Uint64) Add(delta uint64) (new uint64) { return atomic.AddUint64(i.value, delta) }
func (i Uint64) Load() (val uint64)            { return atomic.LoadUint64(i.value) }
func (i Uint64) Store(val uint64)              { atomic.StoreUint64(i.value, val) }
func (i Uint64) Swap(new uint64) (old uint64)  { return atomic.SwapUint64(i.value, new) }
func (i Uint64) CompareAndSwap(old, new uint64) (swapped bool) {
	return atomic.CompareAndSwapUint64(i.value, old, new)
}
