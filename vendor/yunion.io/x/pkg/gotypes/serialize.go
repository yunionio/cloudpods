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

var ISerializableType reflect.Type
var serializableAllocators map[reflect.Type]FuncSerializableAllocator
var ErrTypeNotSerializable error

func init() {
	ISerializableType = reflect.TypeOf((*ISerializable)(nil)).Elem()
	serializableAllocators = make(map[reflect.Type]FuncSerializableAllocator)
	ErrTypeNotSerializable = errors.New("Type not serializable")
}

func RegisterSerializable(valType reflect.Type, alloc FuncSerializableAllocator) {
	serializableAllocators[valType] = alloc
}

func IsSerializable(objType reflect.Type) bool {
	_, ok := serializableAllocators[objType]
	return ok
}

func NewSerializable(objType reflect.Type) (ISerializable, error) {
	deserFunc, ok := serializableAllocators[objType]
	if !ok {
		return nil, ErrTypeNotSerializable
	} else {
		return deserFunc(), nil
	}
}
