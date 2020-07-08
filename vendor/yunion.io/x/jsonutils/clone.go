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

	"yunion.io/x/pkg/sortedmap"
	"yunion.io/x/pkg/utils"
)

func (this *JSONDict) Copy(excludes ...string) *JSONDict {
	return this.CopyExcludes(excludes...)
}

func (this *JSONDict) CopyExcludes(excludes ...string) *JSONDict {
	dict := NewDict()
	for iter := sortedmap.NewIterator(this.data); iter.HasMore(); iter.Next() {
		k, v := iter.Get()
		exists, _ := utils.InStringArray(k, excludes)
		if !exists {
			dict.Set(k, v.(JSONObject))
		}
	}
	return dict
}

func (this *JSONDict) CopyIncludes(includes ...string) *JSONDict {
	dict := NewDict()
	for iter := sortedmap.NewIterator(this.data); iter.HasMore(); iter.Next() {
		k, v := iter.Get()
		exists, _ := utils.InStringArray(k, includes)
		if exists {
			dict.Set(k, v.(JSONObject))
		}
	}
	return dict
}

func (this *JSONArray) Copy() *JSONArray {
	arr := NewArray()
	for _, v := range this.data {
		arr.data = append(arr.data, v)
	}
	return arr
}

func (this *JSONString) DeepCopy() interface{} {
	return DeepCopy(this)
}

func (this *JSONInt) DeepCopy() interface{} {
	return DeepCopy(this)
}

func (this *JSONFloat) DeepCopy() interface{} {
	return DeepCopy(this)
}

func (this *JSONBool) DeepCopy() interface{} {
	return DeepCopy(this)
}

func (this *JSONArray) DeepCopy() interface{} {
	return DeepCopy(this)
}

func (this *JSONDict) DeepCopy() interface{} {
	return DeepCopy(this)
}

func DeepCopy(obj JSONObject) JSONObject {
	if obj == nil || reflect.ValueOf(obj).IsNil() {
		return nil
	}
	switch v := obj.(type) {
	case *JSONString:
		vc := *v
		return &vc
	case *JSONInt:
		vc := *v
		return &vc
	case *JSONFloat:
		vc := *v
		return &vc
	case *JSONBool:
		vc := *v
		return &vc
	case *JSONArray:
		elemsC := make([]JSONObject, v.Length())
		for i := 0; i < v.Length(); i++ {
			elem, _ := v.GetAt(i)
			elemC := DeepCopy(elem)
			elemsC[i] = elemC
		}
		vc := NewArray(elemsC...)
		return vc
	case *JSONDict:
		vc := NewDict()
		m, _ := v.GetMap()
		for mk, mv := range m {
			mvc := DeepCopy(mv)
			vc.Set(mk, mvc)
		}
		return vc
	}
	return nil
}
