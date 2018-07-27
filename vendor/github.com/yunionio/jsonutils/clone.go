package jsonutils

import (
	"github.com/yunionio/pkg/utils"
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
