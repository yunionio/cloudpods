package stm

type VarValue interface {
	Set(any) VarValue
	Get() any
	Changed(VarValue) bool
}

type version uint64

type versionedValue[T any] struct {
	value   T
	version version
}

func (me versionedValue[T]) Set(newValue any) VarValue {
	return versionedValue[T]{
		value:   newValue.(T),
		version: me.version + 1,
	}
}

func (me versionedValue[T]) Get() any {
	return me.value
}

func (me versionedValue[T]) Changed(other VarValue) bool {
	return me.version != other.(versionedValue[T]).version
}

type customVarValue[T any] struct {
	value   T
	changed func(T, T) bool
}

var _ VarValue = customVarValue[struct{}]{}

func (me customVarValue[T]) Changed(other VarValue) bool {
	return me.changed(me.value, other.(customVarValue[T]).value)
}

func (me customVarValue[T]) Set(newValue any) VarValue {
	return customVarValue[T]{
		value:   newValue.(T),
		changed: me.changed,
	}
}

func (me customVarValue[T]) Get() any {
	return me.value
}
