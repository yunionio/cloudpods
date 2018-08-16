package gotypes

import (
	"errors"
	"reflect"
)

type ISerializable interface {
	String() string
	// FromString(str string) error
	// Equals(obj ISerializable) bool
	IsZero() bool
}

type FuncSerializableAllocator func() ISerializable

var (
	ISerializableType      = reflect.TypeOf((*ISerializable)(nil)).Elem()
	serializableAllocators = map[reflect.Type]FuncSerializableAllocator{}
	ErrTypeNotSerializable = errors.New("Type not serializable")
)

// RegisterSerializable registers an allocator func for the specified serializable type.
//
// This is intended to be used when you have multiple implmenetations of an
// interface and you want to use only one of them to cover them all.
// TokenCredential and SSimpleToken is such a case.
func RegisterSerializable(valType reflect.Type, alloc FuncSerializableAllocator) {
	if !IsSerializable(valType) {
		panic(valType.String() + " does not implement ISerializable")
	}
	serializableAllocators[valType] = alloc
}

func IsSerializable(valType reflect.Type) bool {
	return valType.Implements(ISerializableType)
}

func NewSerializable(objType reflect.Type) (ISerializable, error) {
	deserFunc, ok := serializableAllocators[objType]
	if ok {
		return deserFunc(), nil
	}
	return nil, ErrTypeNotSerializable
}
