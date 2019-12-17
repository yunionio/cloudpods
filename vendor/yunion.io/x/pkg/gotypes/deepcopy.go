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
	"reflect"
)

type IDeepCopy interface {
	DeepCopy() interface{}
}

type DeepCopyFlags uintptr

// DeepCopy copies the passed argument recursively
//
// There are a few constraint
//
//  - It cannot handle un-exported fields
//  - It cannot handle circular references.  Doing so will cause runtime stack
//    overflow panic
func DeepCopy(v interface{}) interface{} {
	rv := reflect.ValueOf(v)
	cpRv := DeepCopyRv(rv)
	cpV := cpRv.Interface()
	return cpV
}

func DeepCopyRv(rv reflect.Value) reflect.Value {
	kind := rv.Kind()
	if kind == reflect.Invalid {
		return reflect.Value{}
	}
	typ := rv.Type()

	// XXX: for copy unexportable filed
	if rv.CanInterface() {
		if rvCopyer, ok := rv.Interface().(IDeepCopy); ok {
			if rv.Kind() == reflect.Ptr && rv.IsNil() {
				return reflect.New(typ).Elem()
			}
			cpRv := rvCopyer.DeepCopy()
			if cpRv == nil {
				return reflect.New(typ).Elem()
			}
			return reflect.ValueOf(cpRv)
		}
	}

	switch kind {
	case reflect.Ptr:
		if rv.IsNil() {
			return reflect.New(typ).Elem()
		}
		elemRv := rv.Elem()
		cpElemRv := DeepCopyRv(elemRv)
		cpRv := reflect.New(typ.Elem())
		cpRv.Elem().Set(cpElemRv)
		return cpRv
	case reflect.Slice, reflect.Array:
		var cpRv reflect.Value
		switch kind {
		case reflect.Slice:
			if rv.IsNil() {
				return reflect.New(typ).Elem()
			}
			cpRv = reflect.MakeSlice(typ, rv.Len(), rv.Cap())
		case reflect.Array:
			cpRv = reflect.New(typ).Elem()
		}
		n := rv.Len()
		for i := 0; i < n; i++ {
			elem := rv.Index(i)
			cpElem := DeepCopyRv(elem)
			cpRvElem := cpRv.Index(i)
			cpRvElem.Set(cpElem)
		}
		return cpRv
	case reflect.Struct:
		cpRv := reflect.New(typ).Elem()
		if typ == TimeType {
			cpRv.Set(rv)
		}
		n := rv.NumField()
		for i := 0; i < n; i++ {
			f := rv.Field(i)
			if f.CanInterface() {
				cpRvF := cpRv.Field(i)
				cpF := DeepCopyRv(f)
				cpRvF.Set(cpF)
			}
		}
		return cpRv
	case reflect.Map:
		if rv.IsNil() {
			return reflect.New(typ).Elem()
		}
		cpRv := reflect.MakeMap(typ)
		mk := rv.MapKeys()
		for _, k := range mk {
			v := rv.MapIndex(k)
			cpK := DeepCopyRv(k)
			cpV := DeepCopyRv(v)
			cpRv.SetMapIndex(cpK, cpV)
		}
		return cpRv
	case reflect.Interface:
		cpRv := reflect.New(typ).Elem()
		if !rv.IsNil() {
			elemRv := rv.Elem()
			cpElemRv := DeepCopyRv(elemRv)
			cpRv.Set(cpElemRv)
		}
		return cpRv
	case reflect.Chan:
		if rv.IsNil() {
			return reflect.New(typ).Elem()
		}
		return reflect.MakeChan(typ.Elem(), rv.Cap())
	default:
		// Invalid, Bool, Int*, Uint*, Float*, Complex*, Func, String
		// TODO UnsafePointer
		cpRv := reflect.New(typ).Elem()
		cpRv.Set(rv)
		return cpRv
	}
	panic("unhandled kind " + kind.String())
}
