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

package jsonutils

import (
	"encoding/json"
	"fmt"
	"reflect"

	"yunion.io/x/pkg/gotypes"
)

func tryStdUnmarshal(s *sJsonUnmarshalSession, jo JSONObject, v reflect.Value, unmarshalFunc func(s *sJsonUnmarshalSession, value reflect.Value) error) error {
	u := IsImplementStdUnmarshaler(v)
	if u != nil {
		return u.UnmarshalJSON([]byte(jo.String()))
	}
	return unmarshalFunc(s, v)
}

func IsImplementStdUnmarshaler(v reflect.Value) json.Unmarshaler {
	return indirectStdUnmarshaler(v)
}

func indirectStdUnmarshaler(v reflect.Value) json.Unmarshaler {
	v0 := v
	haveAddr := false

	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		haveAddr = true
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && e.Elem().Kind() == reflect.Ptr {
				haveAddr = false
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		// Prevent infinite loop if v is an interface pointing to its own address:
		//    var v interface{}
		//    v = &v
		if v.Elem().Kind() == reflect.Interface && v.Elem().Elem() == v {
			v = v.Elem()
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if v.Type().NumMethod() > 0 && v.CanInterface() {
			if u, ok := v.Interface().(json.Unmarshaler); ok {
				return u
			}
		}

		if haveAddr {
			v = v0 // restore original value after round-trip Value.Addr().Elem()
			haveAddr = false
		} else {
			v = v.Elem()
		}
	}

	return nil
}

func IsImplementStdMarshaler(v reflect.Value) json.Marshaler {
	return indirectStdMarshaler(v)
}

func indirectStdMarshaler(v reflect.Value) json.Marshaler {
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return nil
	}
	if v.Type().NumMethod() > 0 && v.CanInterface() {
		if u, ok := v.Interface().(json.Marshaler); ok {
			return u
		}
	}
	return nil
}

func tryStdMarshal(v reflect.Value, marshalFunc func(v reflect.Value) JSONObject) JSONObject {
	if v.Type() != gotypes.TimeType {
		m := IsImplementStdMarshaler(v)
		if m != nil {
			data, err := m.MarshalJSON()
			if err != nil {
				panic(fmt.Sprintf("MarshalJSON of %q error: %v", v.String(), err))
			}
			jo, err := Parse(data)
			if err != nil {
				panic(fmt.Sprintf("Parse data %q to json of %q error: %v", data, v.String(), err))
			}
			return jo
		}
	}
	return marshalFunc(v)
}
