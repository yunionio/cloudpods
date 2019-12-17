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
	"reflect"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
)

var (
	JSONDictType      reflect.Type
	JSONArrayType     reflect.Type
	JSONStringType    reflect.Type
	JSONIntType       reflect.Type
	JSONFloatType     reflect.Type
	JSONBoolType      reflect.Type
	JSONDictPtrType   reflect.Type
	JSONArrayPtrType  reflect.Type
	JSONStringPtrType reflect.Type
	JSONIntPtrType    reflect.Type
	JSONFloatPtrType  reflect.Type
	JSONBoolPtrType   reflect.Type
	JSONObjectType    reflect.Type
)

func init() {
	JSONDictType = reflect.TypeOf(JSONDict{})
	JSONArrayType = reflect.TypeOf(JSONArray{})
	JSONStringType = reflect.TypeOf(JSONString{})
	JSONIntType = reflect.TypeOf(JSONInt{})
	JSONFloatType = reflect.TypeOf(JSONFloat{})
	JSONBoolType = reflect.TypeOf(JSONBool{})
	JSONDictPtrType = reflect.TypeOf(&JSONDict{})
	JSONArrayPtrType = reflect.TypeOf(&JSONArray{})
	JSONStringPtrType = reflect.TypeOf(&JSONString{})
	JSONIntPtrType = reflect.TypeOf(&JSONInt{})
	JSONFloatPtrType = reflect.TypeOf(&JSONFloat{})
	JSONBoolPtrType = reflect.TypeOf(&JSONBool{})
	JSONObjectType = reflect.TypeOf((*JSONObject)(nil)).Elem()

	gotypes.RegisterSerializable(JSONObjectType, func() gotypes.ISerializable {
		return nil
	})

	gotypes.RegisterSerializable(JSONDictPtrType, func() gotypes.ISerializable {
		return NewDict()
	})

	gotypes.RegisterSerializable(JSONArrayPtrType, func() gotypes.ISerializable {
		return NewArray()
	})
}

func JSONDeserialize(objType reflect.Type, strVal string) (gotypes.ISerializable, error) {
	objPtr, err := gotypes.NewSerializable(objType)
	if err != nil {
		return nil, errors.Wrap(err, "gotypes.NewSerializable")
	}
	json, err := ParseString(strVal)
	if err != nil {
		return nil, errors.Wrap(err, "ParseString")
	}
	if objPtr == nil {
		return json, nil
	}
	err = json.Unmarshal(objPtr)
	if err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}
	objPtr = gotypes.Transform(objType, objPtr)
	return objPtr, nil
}
