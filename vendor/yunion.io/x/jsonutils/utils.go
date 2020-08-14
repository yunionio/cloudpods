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
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"
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

func GetStringArray(o JSONObject, key ...string) ([]string, error) {
	arr, err := o.GetArray(key...)
	if err != nil {
		return nil, err
	}
	return JSONArray2StringArray(arr), nil
}

func NewTimeString(tm time.Time) *JSONString {
	return NewString(tm.UTC().Format("2006-01-02T15:04:05Z"))
}

func GetQueryStringArray(query JSONObject, key string) []string {
	if query == nil {
		return nil
	}
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
		return errors.Wrap(err, "data.GetMap") //fmt.Errorf("fail to convert input to map")
	}
	for _, f := range fields {
		jsonVal, ok := jsonMap[f]
		if !ok {
			return errors.Wrap(ErrMissingInputField, f)
		}
		if jsonVal == JSONNull {
			return errors.Wrap(ErrNilInputField, f)
		}
	}
	return nil
}

func GetAnyString(json JSONObject, keys []string) string {
	val, _ := GetAnyString2(json, keys)
	return val
}

func GetAnyString2(json JSONObject, keys []string) (string, string) {
	if json == nil {
		return "", ""
	}
	for _, key := range keys {
		val, _ := json.GetString(key)
		if len(val) > 0 {
			return val, key
		}
	}
	return "", ""
}

func GetArrayOfPrefix(json JSONObject, prefix string) []JSONObject {
	if json == nil {
		return nil
	}

	retArray := make([]JSONObject, 0)
	idx := 0
	for {
		obj, _ := json.Get(fmt.Sprintf("%s.%d", prefix, idx))
		if obj == nil || obj == JSONNull {
			break
		}
		retArray = append(retArray, obj)
		idx += 1
	}
	return retArray
}
