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
	"strconv"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/sortedmap"
	"yunion.io/x/pkg/util/timeutils"
)

type JSONPair struct {
	key string
	val JSONObject
}

func NewDict(objs ...JSONPair) *JSONDict {
	dict := JSONDict{data: sortedmap.NewSortedMapWithCapa(len(objs))}
	for _, o := range objs {
		dict.Set(o.key, o.val)
	}
	return &dict
}

func NewArray(objs ...JSONObject) *JSONArray {
	arr := JSONArray{data: make([]JSONObject, 0, len(objs))}
	for _, o := range objs {
		arr.data = append(arr.data, o)
	}
	return &arr
}

func NewString(val string) *JSONString {
	return &JSONString{data: val}
}

func NewInt(val int64) *JSONInt {
	return &JSONInt{data: val}
}

//deprecated
func NewFloat(val float64) *JSONFloat {
	return &JSONFloat{data: val, bit: 64}
}

func NewFloat64(val float64) *JSONFloat {
	return &JSONFloat{data: val, bit: 64}
}

func NewFloat32(val float32) *JSONFloat {
	return &JSONFloat{data: float64(val), bit: 32}
}

func NewBool(val bool) *JSONBool {
	if val {
		return JSONTrue
	}
	return JSONFalse
}

func (this *JSONDict) Set(key string, value JSONObject) {
	this.data = sortedmap.Add(this.data, key, value)
}

func (this *JSONDict) Remove(key string) bool {
	return this.remove(key, true)
}

func (this *JSONDict) RemoveIgnoreCase(key string) bool {
	someRemoved := false
	for {
		removed := this.remove(key, false)
		if !removed {
			break
		}
		if !someRemoved {
			someRemoved = true
		}
	}
	return someRemoved
}

func (this *JSONDict) remove(key string, caseSensitive bool) bool {
	exist := false
	if !caseSensitive {
		this.data, _, exist = sortedmap.DeleteIgnoreCase(this.data, key)
	} else {
		this.data, exist = sortedmap.Delete(this.data, key)
	}
	return exist
}

func (this *JSONDict) Add(o JSONObject, keys ...string) error {
	obj := this
	for i := 0; i < len(keys); i++ {
		if i == len(keys)-1 {
			obj.Set(keys[i], o)
		} else {
			o, ok := obj.data.Get(keys[i])
			if !ok || o == JSONNull {
				obj.Set(keys[i], NewDict())
				o, ok = obj.data.Get(keys[i])
			}
			if ok {
				obj, ok = o.(*JSONDict)
				if !ok {
					return ErrInvalidJsonDict
				}
			} else {
				return ErrJsonDictFailInsert
			}
		}
	}
	return nil
}

func (this *JSONArray) SetAt(idx int, obj JSONObject) {
	this.data[idx] = obj
}

func (this *JSONArray) Add(objs ...JSONObject) {
	for _, o := range objs {
		this.data = append(this.data, o)
	}
}

func (this *JSONValue) Contains(keys ...string) bool {
	return false
}

func (this *JSONValue) ContainsIgnoreCases(keys ...string) bool {
	return false
}

func (this *JSONValue) Get(keys ...string) (JSONObject, error) {
	return nil, ErrUnsupported
}

func (this *JSONValue) GetIgnoreCases(keys ...string) (JSONObject, error) {
	return nil, ErrUnsupported
}

func (this *JSONValue) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		return "", ErrOutOfKeyRange
	}
	return this.String(), nil
}

func (this *JSONValue) GetAt(i int, keys ...string) (JSONObject, error) {
	return nil, ErrUnsupported
}

func (this *JSONValue) Int(keys ...string) (int64, error) {
	return 0, ErrUnsupported
}

func (this *JSONValue) Float(keys ...string) (float64, error) {
	return 0.0, ErrUnsupported
}

func (this *JSONValue) Bool(keys ...string) (bool, error) {
	return false, ErrUnsupported
}

func (this *JSONValue) GetMap(keys ...string) (map[string]JSONObject, error) {
	return nil, ErrUnsupported
}

func (this *JSONValue) GetArray(keys ...string) ([]JSONObject, error) {
	return nil, ErrUnsupported
}

func (this *JSONDict) Contains(keys ...string) bool {
	obj, err := this._get(true, keys)
	if err == nil && obj != nil {
		return true
	}
	return false
}

func (this *JSONDict) ContainsIgnoreCases(keys ...string) bool {
	obj, err := this._get(false, keys)
	if err == nil && obj != nil {
		return true
	}
	return false
}

func (this *JSONDict) Get(keys ...string) (JSONObject, error) {
	return this._get(true, keys)
}

func (this *JSONDict) GetIgnoreCases(keys ...string) (JSONObject, error) {
	return this._get(false, keys)
}

func (this *JSONDict) _get(caseSensitive bool, keys []string) (JSONObject, error) {
	if len(keys) == 0 {
		return this, nil
	}
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		var val interface{}
		var ok bool
		if caseSensitive {
			val, ok = this.data.Get(key)
		} else {
			val, _, ok = this.data.GetIgnoreCase(key)
		}
		if ok {
			if i == len(keys)-1 {
				return val.(JSONObject), nil
			} else {
				this, ok = val.(*JSONDict)
				if !ok {
					return nil, ErrInvalidJsonDict
				}
			}
		} else {
			return nil, ErrJsonDictKeyNotFound
		}
	}
	return nil, ErrOutOfKeyRange
}

func (this *JSONDict) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		obj, err := this.Get(keys...)
		if err != nil {
			return "", errors.Wrap(err, "Get")
		}
		return obj.GetString()
	} else {
		return this.String(), nil
	}
}

func (this *JSONDict) GetMap(keys ...string) (map[string]JSONObject, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return nil, errors.Wrap(err, "Get")
	}
	dict, ok := obj.(*JSONDict)
	if !ok {
		return nil, ErrInvalidJsonDict
	}
	// allocate a map to hold the return map
	ret := make(map[string]JSONObject, len(this.data))
	for iter := sortedmap.NewIterator(dict.data); iter.HasMore(); iter.Next() {
		k, v := iter.Get()
		ret[k] = v.(JSONObject)
	}
	return ret, nil
}

func (this *JSONArray) GetAt(i int, keys ...string) (JSONObject, error) {
	if len(keys) > 0 {
		return nil, ErrOutOfKeyRange //  fmt.Errorf("Out of key range: %s", keys)
	}
	if i < 0 {
		i = len(this.data) + i
	}
	if i >= 0 && i < len(this.data) {
		return this.data[i], nil
	} else {
		return nil, ErrOutOfIndexRange // fmt.Errorf("Out of range GetAt(%d)", i)
	}
}

func (this *JSONArray) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		return "", ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	return this.String(), nil
}

func (this *JSONDict) GetAt(i int, keys ...string) (JSONObject, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return nil, errors.Wrap(err, "Get")
	}
	arr, ok := obj.(*JSONArray)
	if !ok {
		return nil, ErrInvalidJsonArray // fmt.Errorf("%s is not an Array", keys)
	}
	return arr.GetAt(i)
}

func (this *JSONArray) GetArray(keys ...string) ([]JSONObject, error) {
	if len(keys) > 0 {
		return nil, ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	// Allocate a new array to hold the return array
	ret := make([]JSONObject, len(this.data))
	for i := range this.data {
		ret[i] = this.data[i]
	}
	return ret, nil
}

func (this *JSONDict) GetArray(keys ...string) ([]JSONObject, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return nil, errors.Wrap(err, "Get")
	}
	if _, ok := obj.(*JSONDict); ok {
		return nil, ErrInvalidJsonArray
	}
	return obj.GetArray()
	/* arr, ok := obj.(*JSONArray)
	   if !ok {
	       return nil, fmt.Errorf("%s is not an Array", keys)
	   }
	   return arr.GetArray() */
}

func _getarray(obj JSONObject, keys ...string) ([]JSONObject, error) {
	if len(keys) > 0 {
		return nil, ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	return []JSONObject{obj}, nil
}

func (this *JSONString) GetArray(keys ...string) ([]JSONObject, error) {
	return _getarray(this, keys...)
}

func (this *JSONInt) GetArray(keys ...string) ([]JSONObject, error) {
	return _getarray(this, keys...)
}

func (this *JSONFloat) GetArray(keys ...string) ([]JSONObject, error) {
	return _getarray(this, keys...)
}

func (this *JSONBool) GetArray(keys ...string) ([]JSONObject, error) {
	return _getarray(this, keys...)
}

func (this *JSONInt) Int(keys ...string) (int64, error) {
	if len(keys) > 0 {
		return 0, ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	return this.data, nil
}

func (this *JSONString) Int(keys ...string) (int64, error) {
	if len(keys) > 0 {
		return 0, ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	val, e := strconv.ParseInt(this.data, 10, 64)
	if e != nil {
		return 0, ErrInvalidJsonInt // fmt.Errorf("Invalid number %s", this.data)
	} else {
		return val, nil
	}
}

func (this *JSONInt) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		return "", ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	return this.String(), nil
}

func (this *JSONDict) Int(keys ...string) (int64, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return 0, errors.Wrap(err, "Get")
	}
	return obj.Int()
	/* jint, ok := obj.(*JSONInt)
	   if ! ok {
	       return 0, fmt.Errorf("%s is not an Int", keys)
	   }
	   return jint.Int() */
}

func (this *JSONFloat) Float(keys ...string) (float64, error) {
	if len(keys) > 0 {
		return 0.0, ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	return this.data, nil
}

func (this *JSONInt) Float(keys ...string) (float64, error) {
	if len(keys) > 0 {
		return 0.0, ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	return float64(this.data), nil
}

func (this *JSONString) Float(keys ...string) (float64, error) {
	if len(keys) > 0 {
		return 0.0, ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	val, err := strconv.ParseFloat(this.data, 64)
	if err != nil {
		return 0.0, ErrInvalidJsonFloat // fmt.Errorf("Not a float %s", this.data)
	} else {
		return val, nil
	}
}

func (this *JSONFloat) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		return "", ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	return this.String(), nil
}

func (this *JSONDict) Float(keys ...string) (float64, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return 0, errors.Wrap(err, "Get")
	}
	return obj.Float()
	/* jfloat, ok := obj.(*JSONFloat)
	   if ! ok {
	       return 0, fmt.Errorf("%s is not a float", keys)
	   }
	   return jfloat.Float() */
}

func (this *JSONBool) Bool(keys ...string) (bool, error) {
	if len(keys) > 0 {
		return false, ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	return this.data, nil
}

func (this *JSONString) Bool(keys ...string) (bool, error) {
	if len(keys) > 0 {
		return false, ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	if strings.EqualFold(this.data, "true") || strings.EqualFold(this.data, "on") || strings.EqualFold(this.data, "yes") || this.data == "1" {
		return true, nil
	} else if strings.EqualFold(this.data, "false") || strings.EqualFold(this.data, "off") || strings.EqualFold(this.data, "no") || this.data == "0" {
		return false, nil
	} else {
		return false, ErrInvalidJsonBoolean // fmt.Errorf("Invalid boolean string %s", this.data)
	}
}

func (this *JSONBool) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		return "", ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	return this.String(), nil
}

func (this *JSONDict) Bool(keys ...string) (bool, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return false, errors.Wrap(err, "Get")
	}
	return obj.Bool()
	/* jbool, ok := obj.(*JSONBool)
	   if ! ok {
	       return false, fmt.Errorf("%s is not a bool", keys)
	   }
	   return jbool.Bool() */
}

func (this *JSONValue) GetTime(keys ...string) (time.Time, error) {
	return time.Time{}, ErrUnsupported // fmt.Errorf("Unsupported operation GetTime")
}

func (this *JSONString) GetTime(keys ...string) (time.Time, error) {
	if len(keys) > 0 {
		return time.Time{}, ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	t, e := timeutils.ParseTimeStr(this.data)
	if e == nil {
		return t, nil
	}
	for _, tf := range []string{time.RFC3339, time.RFC1123, time.UnixDate,
		time.RFC822,
	} {
		t, e := time.Parse(tf, this.data)
		if e == nil {
			return t, nil
		}
	}
	return this.JSONValue.GetTime()
}

func (this *JSONString) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		return "", ErrOutOfKeyRange // fmt.Errorf("Out of key range: %s", keys)
	}
	return this.data, nil
}

func (this *JSONDict) GetTime(keys ...string) (time.Time, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "Get")
	}
	str, ok := obj.(*JSONString)
	if !ok {
		return time.Time{}, ErrInvalidJsonString // fmt.Errorf("%s is not a string", keys)
	}
	return str.GetTime()
}

/*
func (this *JSONDict) GetIgnoreCases(key ...string) (JSONObject, bool) {
    lkey := strings.ToLower(key)
    for k, v := range this.data {
        if strings.ToLower(k) == lkey {
            return v, true
        }
    }
    return nil, false
}*/
