package jsonutils

import (
	"fmt"
	"time"
)

func NewStringArray(arr []string) *JSONArray {
	ret := NewArray()
	for _, a := range arr {
		ret.Add(NewString(a))
	}
	return ret
}

func (this *JSONArray) GetStringArray() []string {
	ret := make([]string, len(this.data))
	for i, obj := range this.data {
		s, e := obj.GetString()
		if e == nil {
			ret[i] = s
		}
	}
	return ret
}

func JSONArray2StringArray(arr []JSONObject) []string {
	ret := make([]string, len(arr))
	for i, o := range arr {
		s, e := o.GetString()
		if e == nil {
			ret[i] = s
		}
	}
	return ret
}

func NewTimeString(tm time.Time) *JSONString {
	return NewString(tm.Format("2006-01-02T15:04:05Z"))
}

func GetQueryStringArray(query JSONObject, key string) []string {
	var arr []string
	searchObj, _ := query.Get(key)
	if searchObj != nil {
		switch searchObj.(type) {
		case *JSONArray:
			searchArr := searchObj.(*JSONArray)
			arr = searchArr.GetStringArray()
		case *JSONString:
			searchText, _ := searchObj.(*JSONString).GetString()
			arr = []string{searchText}
		case *JSONDict:
			arr = make([]string, 0)
			idx := 0
			for {
				searchText, err := searchObj.GetString(fmt.Sprintf("%d", idx))
				if err != nil {
					break
				}
				arr = append(arr, searchText)
				idx += 1

			}
		}
	}
	return arr
}

func CheckRequiredFields(data JSONObject, fields []string) error {
	jsonMap, err := data.GetMap()
	if err != nil {
		return fmt.Errorf("fail to convert input to map")
	}
	for _, f := range fields {
		jsonVal, ok := jsonMap[f]
		if !ok {
			return fmt.Errorf("missing input field %s", f)
		}
		if jsonVal == JSONNull {
			return fmt.Errorf("input field %s is null", f)
		}
	}
	return nil
}
