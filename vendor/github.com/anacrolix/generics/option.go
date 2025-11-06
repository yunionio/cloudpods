package generics

type Option[V any] struct {
	// Value must be zeroed when Ok is false for deterministic comparability.
	Value V
	// bool is the smallest type, so putting it at the end increases the chance it can be packed
	// with Value.
	Ok bool
}

func (me Option[V]) UnwrapOrZeroValue() (_ V) {
	if me.Ok {
		return me.Value
	}
	return
}

func (me *Option[V]) UnwrapPtr() *V {
	if !me.Ok {
		panic("not set")
	}
	return &me.Value
}

func (me Option[V]) Unwrap() V {
	if !me.Ok {
		panic("not set")
	}
	return me.Value
}

// Deprecated: Use option.AndThen
func (me Option[V]) AndThen(f func(V) Option[V]) Option[V] {
	if me.Ok {
		return f(me.Value)
	}
	return me
}

func (me Option[V]) UnwrapOr(or V) V {
	if me.Ok {
		return me.Value
	} else {
		return or
	}
}

func (me *Option[V]) Set(v V) (prev Option[V]) {
	prev = *me
	me.Ok = true
	me.Value = v
	return
}

func (me *Option[V]) SetNone() {
	me.Ok = false
	me.Value = ZeroValue[V]()
}

func (me *Option[V]) SetFromTuple(v V, ok bool) {
	*me = OptionFromTuple(v, ok)
}

func (me *Option[V]) SetSomeZeroValue() {
	me.Ok = true
	me.Value = ZeroValue[V]()
}

func Some[V any](value V) Option[V] {
	return Option[V]{Ok: true, Value: value}
}

func None[V any]() Option[V] {
	return Option[V]{}
}

func OptionFromTuple[T any](t T, ok bool) Option[T] {
	if ok {
		return Some(t)
	} else {
		return None[T]()
	}
}
