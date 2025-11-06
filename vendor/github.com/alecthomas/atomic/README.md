# Type-safe atomic values for Go

One issue with Go's sync/atomic package is that there is no guarantee from the
type system that operations on an integer value will be applied through the
sync/atomic functions. This package solves that and introduces two type-safe
interfaces for use with integer and non-integer atomic values.

The first interface is for any value:

```go
// Value represents a value that can be atomically loaded or replaced.
type Value[T any] interface {
	// Load value atomically.
	Load() T
	// Store value atomically.
	Store(value T)
	// Swap the previous value with the new value atomically.
	Swap(new T) (old T)
	// CompareAndSwap the previous value with new if its value is "old".
	CompareAndSwap(old, new T) (swapped bool)
}
```

The second interface is a `Value[T]` constrained to the 32 and 64 bit integer
types and adds a single `Add()` method:

```go
// Int expresses atomic operations on signed or unsigned integer values.
type Int[T atomicint] interface {
	Value[T]
	// Add a value and return the new result.
	Add(delta T) (new T)
}

```

## Performance

```
BenchmarkInt64Add
BenchmarkInt64Add-8           	174217112	        6.887 ns/op
BenchmarkIntInterfaceAdd
BenchmarkIntInterfaceAdd-8    	174129980	        6.889 ns/op
BenchmarkStdlibInt64Add
BenchmarkStdlibInt64Add-8     	174152660	        6.887 ns/op
BenchmarkInterfaceStore
BenchmarkInterfaceStore-8     	16015668	       76.17 ns/op
BenchmarkValueStore
BenchmarkValueStore-8         	16155405	       75.03 ns/op
BenchmarkStdlibValueStore
BenchmarkStdlibValueStore-8   	16391035	       74.85 ns/op
```