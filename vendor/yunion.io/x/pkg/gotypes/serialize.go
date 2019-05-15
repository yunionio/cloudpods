// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

type FuncSerializableTransformer func(ISerializable) ISerializable

var (
	ISerializableType        = reflect.TypeOf((*ISerializable)(nil)).Elem()
	serializableAllocators   = map[reflect.Type]FuncSerializableAllocator{}
	serializableTransformers = map[reflect.Type][]FuncSerializableTransformer{}
	ErrTypeNotSerializable   = errors.New("Type not serializable")
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
	if _, ok := serializableAllocators[valType]; ok {
		panic(valType.String() + " has been registered, might need to register a transformer")
	}
	serializableAllocators[valType] = alloc
}

func RegisterSerializableTransformer(valType reflect.Type, trans FuncSerializableTransformer) {
	if !IsSerializable(valType) {
		panic(valType.String() + " does not implement ISerializable")
	}
	if _, ok := serializableTransformers[valType]; !ok {
		serializableTransformers[valType] = make([]FuncSerializableTransformer, 0)
	}
	serializableTransformers[valType] = append(serializableTransformers[valType], trans)
}

func IsSerializable(valType reflect.Type) bool {
	return valType.Implements(ISerializableType)
}

func NewSerializable(objType reflect.Type) (ISerializable, error) {
	deserFunc, ok := serializableAllocators[objType]
	if !ok {
		return nil, ErrTypeNotSerializable
	}
	retVal := deserFunc()
	return retVal, nil
}

func Transform(objType reflect.Type, retVal ISerializable) ISerializable {
	transFuncs, ok := serializableTransformers[objType]
	if ok {
		for i := 0; i < len(transFuncs); i += 1 {
			retVal = transFuncs[i](retVal)
		}
	}
	return retVal
}
