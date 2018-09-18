package jsonutils

import (
	"yunion.io/x/pkg/utils"
)

func (this *JSONDict) Copy(excludes ...string) *JSONDict {
	return this.CopyExcludes(excludes...)
}

func (this *JSONDict) CopyExcludes(excludes ...string) *JSONDict {
	dict := NewDict()
	for k, v := range this.data {
		exists, _ := utils.InStringArray(k, excludes)
		if !exists {
			dict.data[k] = v
		}
	}
	return dict
}

func (this *JSONDict) CopyIncludes(includes ...string) *JSONDict {
	dict := NewDict()
	for k, v := range this.data {
		exists, _ := utils.InStringArray(k, includes)
		if exists {
			dict.data[k] = v
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

func DeepCopy(obj JSONObject) JSONObject {
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
