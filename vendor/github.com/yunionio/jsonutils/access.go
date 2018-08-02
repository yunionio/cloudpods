package jsonutils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/yunionio/pkg/util/timeutils"
)

type JSONPair struct {
	key string
	val JSONObject
}

func NewDict(objs ...JSONPair) *JSONDict {
	dict := JSONDict{data: make(map[string]JSONObject)}
	for _, o := range objs {
		dict.data[o.key] = o.val
	}
	return &dict
}

func NewArray(objs ...JSONObject) *JSONArray {
	arr := JSONArray{data: make([]JSONObject, 0)}
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

func NewFloat(val float64) *JSONFloat {
	return &JSONFloat{data: val}
}

func NewBool(val bool) *JSONBool {
	if val {
		return JSONTrue
	}
	return JSONFalse
}

func (this *JSONDict) Set(key string, value JSONObject) {
	this.data[key] = value
}

func (this *JSONDict) Remove(key string, caseSensitive bool) bool {
	_, rk, ok := dictGet(this.data, key, caseSensitive)
	if ok {
		delete(this.data, rk)
		return true
	} else {
		return false
	}
}

func (this *JSONDict) Add(o JSONObject, keys ...string) error {
	var obj *JSONDict = this
	for i := 0; i < len(keys); i++ {
		if i == len(keys)-1 {
			obj.data[keys[i]] = o
		} else {
			o, ok := obj.data[keys[i]]
			if !ok {
				obj.data[keys[i]] = NewDict()
				o, ok = obj.data[keys[i]]
			}
			if ok {
				obj, ok = o.(*JSONDict)
				if !ok {
					return fmt.Errorf("%s is not a JSONDict", keys[:i])
				}
			} else {
				return fmt.Errorf("Fail to insert %s", keys[i])
			}
		}
	}
	return nil
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
	return nil, fmt.Errorf("Unsupport operation Get")
}

func (this *JSONValue) GetIgnoreCases(keys ...string) (JSONObject, error) {
	return nil, fmt.Errorf("Unsupport operation Get")
}

func (this *JSONValue) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		return "", fmt.Errorf("Out of key range: %s", keys)
	}
	return this.String(), nil
}

func (this *JSONValue) GetAt(i int, keys ...string) (JSONObject, error) {
	return nil, fmt.Errorf("Unsupport operation GetAt")
}

func (this *JSONValue) Int(keys ...string) (int64, error) {
	return 0, fmt.Errorf("Unsupport operation Int")
}

func (this *JSONValue) Float(keys ...string) (float64, error) {
	return 0.0, fmt.Errorf("Unsupport operation Float")
}

func (this *JSONValue) Bool(keys ...string) (bool, error) {
	return false, fmt.Errorf("Unsupport operation Bool")
}

func (this *JSONValue) GetMap(keys ...string) (map[string]JSONObject, error) {
	return make(map[string]JSONObject), fmt.Errorf("Unsupport operation GetMap")
}

func (this *JSONValue) GetArray(keys ...string) ([]JSONObject, error) {
	return nil, fmt.Errorf("Unsupport operation GetMap")
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

func dictGet(dict map[string]JSONObject, key string, caseSensitive bool) (JSONObject, string, bool) {
	if caseSensitive {
		v, ok := dict[key]
		return v, key, ok
	} else {
		for k, v := range dict {
			if strings.EqualFold(k, key) {
				return v, k, true
			}
		}
		return nil, "", false
	}
}

func (this *JSONDict) _get(caseSensitive bool, keys []string) (JSONObject, error) {
	if len(keys) == 0 {
		return this, nil
	}
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		val, _, ok := dictGet(this.data, key, caseSensitive)
		if ok {
			if i == len(keys)-1 {
				return val, nil
			} else {
				this, ok = val.(*JSONDict)
				if !ok {
					return nil, fmt.Errorf("%s is not a Dict", keys[:i])
				}
			}
		} else {
			return nil, fmt.Errorf("No such key %s", key)
		}
	}
	return nil, fmt.Errorf("Out of range key %s", keys)
}

func (this *JSONDict) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		obj, err := this.Get(keys...)
		if err != nil {
			return "", err
		}
		return obj.GetString()
	} else {
		return this.String(), nil
	}
}

func (this *JSONDict) GetMap(keys ...string) (map[string]JSONObject, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return nil, err
	}
	dict, ok := obj.(*JSONDict)
	if !ok {
		return nil, fmt.Errorf("%s is not a Dict", keys)
	}
	return dict.data, nil
}

func (this *JSONArray) GetAt(i int, keys ...string) (JSONObject, error) {
	if len(keys) > 0 {
		return nil, fmt.Errorf("Out of key range: %s", keys)
	}
	if i < 0 {
		i = len(this.data) + i
	}
	if i >= 0 && i < len(this.data) {
		return this.data[i], nil
	} else {
		return nil, fmt.Errorf("Out of range GetAt(%d)", i)
	}
}

func (this *JSONArray) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		return "", fmt.Errorf("Out of key range: %s", keys)
	}
	return this.String(), nil
}

func (this *JSONDict) GetAt(i int, keys ...string) (JSONObject, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return nil, err
	}
	arr, ok := obj.(*JSONArray)
	if !ok {
		return nil, fmt.Errorf("%s is not an Array", keys)
	}
	return arr.GetAt(i)
}

func (this *JSONArray) GetArray(keys ...string) ([]JSONObject, error) {
	if len(keys) > 0 {
		return make([]JSONObject, 0), fmt.Errorf("Out of key range: %s", keys)
	}
	return this.data, nil
}

func (this *JSONDict) GetArray(keys ...string) ([]JSONObject, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("Out of key range: %s", keys)
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
		return 0, fmt.Errorf("Out of key range: %s", keys)
	}
	return this.data, nil
}

func (this *JSONString) Int(keys ...string) (int64, error) {
	if len(keys) > 0 {
		return 0, fmt.Errorf("Out of key range: %s", keys)
	}
	val, e := strconv.ParseInt(this.data, 10, 64)
	if e != nil {
		return 0, fmt.Errorf("Invalid number %s", this.data)
	} else {
		return val, nil
	}
}

func (this *JSONInt) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		return "", fmt.Errorf("Out of key range: %s", keys)
	}
	return this.String(), nil
}

func (this *JSONDict) Int(keys ...string) (int64, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return 0, err
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
		return 0.0, fmt.Errorf("Out of key range: %s", keys)
	}
	return this.data, nil
}

func (this *JSONString) Float(keys ...string) (float64, error) {
	if len(keys) > 0 {
		return 0.0, fmt.Errorf("Out of key range: %s", keys)
	}
	val, err := strconv.ParseFloat(this.data, 64)
	if err != nil {
		return 0.0, fmt.Errorf("Not a float %s", this.data)
	} else {
		return val, nil
	}
}

func (this *JSONFloat) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		return "", fmt.Errorf("Out of key range: %s", keys)
	}
	return this.String(), nil
}

func (this *JSONDict) Float(keys ...string) (float64, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return 0, err
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
		return false, fmt.Errorf("Out of key range: %s", keys)
	}
	return this.data, nil
}

func (this *JSONString) Bool(keys ...string) (bool, error) {
	if len(keys) > 0 {
		return false, fmt.Errorf("Out of key range: %s", keys)
	}
	if strings.EqualFold(this.data, "true") || strings.EqualFold(this.data, "on") || strings.EqualFold(this.data, "yes") || this.data == "1" {
		return true, nil
	} else if strings.EqualFold(this.data, "false") || strings.EqualFold(this.data, "off") || strings.EqualFold(this.data, "no") || this.data == "0" {
		return false, nil
	} else {
		return false, fmt.Errorf("Invalid boolean string %s", this.data)
	}
}

func (this *JSONBool) GetString(keys ...string) (string, error) {
	if len(keys) > 0 {
		return "", fmt.Errorf("Out of key range: %s", keys)
	}
	return this.String(), nil
}

func (this *JSONDict) Bool(keys ...string) (bool, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return false, err
	}
	return obj.Bool()
	/* jbool, ok := obj.(*JSONBool)
	   if ! ok {
	       return false, fmt.Errorf("%s is not a bool", keys)
	   }
	   return jbool.Bool() */
}

func (this *JSONValue) GetTime(keys ...string) (time.Time, error) {
	return time.Time{}, fmt.Errorf("Unsupported operation GetTime")
}

func (this *JSONString) GetTime(keys ...string) (time.Time, error) {
	if len(keys) > 0 {
		return time.Time{}, fmt.Errorf("Out of key range: %s", keys)
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
		return "", fmt.Errorf("Out of key range: %s", keys)
	}
	return this.data, nil
}

func (this *JSONDict) GetTime(keys ...string) (time.Time, error) {
	obj, err := this.Get(keys...)
	if err != nil {
		return time.Time{}, err
	}
	str, ok := obj.(*JSONString)
	if !ok {
		return time.Time{}, fmt.Errorf("%s is not a string", keys)
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
