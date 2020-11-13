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

/**
jsonutils.Marshal

Convert any object to JSONObject

*/

import (
	"fmt"
	"reflect"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/util/timeutils"
)

func marshalSlice(val reflect.Value, info *reflectutils.SStructFieldInfo) JSONObject {
	if val.Len() == 0 && info != nil && info.OmitEmpty {
		return JSONNull
	}
	objs := make([]JSONObject, val.Len())
	for i := 0; i < val.Len(); i += 1 {
		objs[i] = marshalValue(val.Index(i), nil)
	}
	arr := NewArray(objs...)
	if info != nil && info.ForceString {
		return NewString(arr.String())
	} else {
		return arr
	}
}

func marshalMap(val reflect.Value, info *reflectutils.SStructFieldInfo) JSONObject {
	keys := val.MapKeys()
	if len(keys) == 0 && info != nil && info.OmitEmpty {
		return JSONNull
	}
	objPairs := make([]JSONPair, 0)
	for i := 0; i < len(keys); i += 1 {
		key := keys[i]
		val := marshalValue(val.MapIndex(key), nil)
		if val != JSONNull {
			objPairs = append(objPairs, JSONPair{key: fmt.Sprintf("%s", key), val: val})
		}
	}
	dict := NewDict(objPairs...)
	if info != nil && info.ForceString {
		return NewString(dict.String())
	} else {
		return dict
	}
}

func marshalStruct(val reflect.Value, info *reflectutils.SStructFieldInfo) JSONObject {
	objPairs := struct2JSONPairs(val)
	if len(objPairs) == 0 && info != nil && info.OmitEmpty {
		return JSONNull
	}
	dict := NewDict(objPairs...)
	if info != nil && info.ForceString {
		return NewString(dict.String())
	} else {
		return dict
	}
}

func findValueByKey(pairs []JSONPair, key string) JSONObject {
	for i := range pairs {
		if pairs[i].key == key {
			return pairs[i].val
		}
	}
	return nil
}

func struct2JSONPairs(val reflect.Value) []JSONPair {
	fields := reflectutils.FetchStructFieldValueSet(val)
	objPairs := make([]JSONPair, 0, len(fields))
	depFields := make(map[string]string)
	for i := 0; i < len(fields); i += 1 {
		jsonInfo := fields[i].Info
		if jsonInfo.Ignore {
			continue
		}
		key := jsonInfo.MarshalName()
		if deprecatedBy, ok := fields[i].Info.Tags[TAG_DEPRECATED_BY]; ok {
			depFields[key] = deprecatedBy
			continue
		}
		val := marshalValue(fields[i].Value, jsonInfo)
		if val != nil && val != JSONNull {
			objPair := JSONPair{key: key, val: val}
			objPairs = append(objPairs, objPair)
		}
	}
	depPairs := make([]JSONPair, 0, len(depFields))
	for depKey, key := range depFields {
		findLoop := false
		for {
			if okey, ok := depFields[key]; ok {
				if okey == depKey {
					// loop detected
					findLoop = true
					break
				}
				key = okey
			} else {
				break
			}
		}
		if findLoop {
			continue
		}
		val := findValueByKey(objPairs, key)
		if val != nil {
			objPair := JSONPair{key: depKey, val: val}
			depPairs = append(depPairs, objPair)
		}
	}
	objPairs = append(objPairs, depPairs...)
	return objPairs
}

func marshalInt64(val int64, info *reflectutils.SStructFieldInfo) JSONObject {
	if val == 0 && info != nil && info.OmitZero {
		return JSONNull
	} else if info != nil && info.ForceString {
		return NewString(fmt.Sprintf("%d", val))
	} else {
		return NewInt(val)
	}
}

func marshalFloat64(val float64, info *reflectutils.SStructFieldInfo, bit int) JSONObject {
	if val == 0.0 && info != nil && info.OmitZero {
		return JSONNull
	} else if info != nil && info.ForceString {
		return NewString(fmt.Sprintf("%f", val))
	} else {
		return NewFloat64(val)
	}
}

func marshalFloat32(val float32, info *reflectutils.SStructFieldInfo, bit int) JSONObject {
	if val == 0.0 && info != nil && info.OmitZero {
		return JSONNull
	} else if info != nil && info.ForceString {
		return NewString(fmt.Sprintf("%f", val))
	} else {
		return NewFloat32(val)
	}
}

func marshalBoolean(val bool, info *reflectutils.SStructFieldInfo) JSONObject {
	if !val && info != nil && info.OmitFalse {
		return JSONNull
	} else if info != nil && info.ForceString {
		return NewString(fmt.Sprintf("%v", val))
	} else {
		if val {
			return JSONTrue
		} else {
			return JSONFalse
		}
	}
}

func marshalTristate(val tristate.TriState, info *reflectutils.SStructFieldInfo) JSONObject {
	if val.IsTrue() {
		return JSONTrue
	} else if val.IsFalse() {
		return JSONFalse
	} else {
		return JSONNull
	}
}

func marshalString(val string, info *reflectutils.SStructFieldInfo) JSONObject {
	if len(val) == 0 && info != nil && info.OmitEmpty {
		return JSONNull
	} else {
		return NewString(val)
	}
}

func marshalTime(val time.Time, info *reflectutils.SStructFieldInfo) JSONObject {
	if val.IsZero() {
		if info != nil && info.OmitEmpty {
			return JSONNull
		}
		return NewString("")
	} else {
		return NewString(timeutils.FullIsoTime(val))
	}
}

func Marshal(obj interface{}) JSONObject {
	if obj == nil {
		return JSONNull
	}
	val := reflect.ValueOf(obj)
	if kind := val.Kind(); val.IsZero() && kind == reflect.Ptr {
		return JSONNull
	}
	objValue := reflect.Indirect(val)
	return marshalValue(objValue, nil)
}

func marshalValue(objValue reflect.Value, info *reflectutils.SStructFieldInfo) JSONObject {
	switch objValue.Type() {
	case JSONDictPtrType, JSONArrayPtrType, JSONBoolPtrType, JSONIntPtrType, JSONFloatPtrType, JSONStringPtrType, JSONObjectType:
		if objValue.IsNil() {
			return JSONNull
		}
		return objValue.Interface().(JSONObject)
	case JSONDictType:
		json, ok := objValue.Interface().(JSONDict)
		if ok {
			if len(json.data) == 0 && info != nil && info.OmitEmpty {
				return JSONNull
			} else {
				return &json
			}
		} else {
			return JSONNull
		}
	case JSONArrayType:
		json, ok := objValue.Interface().(JSONArray)
		if ok {
			if len(json.data) == 0 && info != nil && info.OmitEmpty {
				return JSONNull
			} else {
				return &json
			}
		} else {
			return JSONNull
		}
	case JSONBoolType:
		json, ok := objValue.Interface().(JSONBool)
		if ok {
			if !json.data && info != nil && info.OmitEmpty {
				return JSONNull
			} else {
				return &json
			}
		} else {
			return JSONNull
		}
	case JSONIntType:
		json, ok := objValue.Interface().(JSONInt)
		if ok {
			if json.data == 0 && info != nil && info.OmitEmpty {
				return JSONNull
			} else {
				return &json
			}
		} else {
			return JSONNull
		}
	case JSONFloatType:
		json, ok := objValue.Interface().(JSONFloat)
		if ok {
			if json.data == 0.0 && info != nil && info.OmitEmpty {
				return JSONNull
			} else {
				return &json
			}
		} else {
			return JSONNull
		}
	case JSONStringType:
		json, ok := objValue.Interface().(JSONString)
		if ok {
			if len(json.data) == 0 && info != nil && info.OmitEmpty {
				return JSONNull
			} else {
				return &json
			}
		} else {
			return JSONNull
		}
	case tristate.TriStateType:
		tri, ok := objValue.Interface().(tristate.TriState)
		if ok {
			return marshalTristate(tri, info)
		} else {
			return JSONNull
		}
	}
	switch objValue.Kind() {
	case reflect.Slice, reflect.Array:
		return marshalSlice(objValue, info)
	case reflect.Struct:
		if objValue.Type() == gotypes.TimeType {
			return marshalTime(objValue.Interface().(time.Time), info)
		} else {
			return marshalStruct(objValue, info)
		}
	case reflect.Map:
		return marshalMap(objValue, info)
	case reflect.String:
		strValue := objValue.Convert(gotypes.StringType)
		return marshalString(strValue.Interface().(string), info)
	case reflect.Bool:
		return marshalBoolean(objValue.Interface().(bool), info)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		intValue := objValue.Convert(gotypes.Int64Type)
		return marshalInt64(intValue.Interface().(int64), info)
	case reflect.Float32:
		floatVal := objValue.Convert(gotypes.Float32Type)
		return marshalFloat32(floatVal.Interface().(float32), info, 32)
	case reflect.Float64:
		floatVal := objValue.Convert(gotypes.Float64Type)
		return marshalFloat64(floatVal.Interface().(float64), info, 64)
	case reflect.Interface, reflect.Ptr:
		if objValue.IsNil() {
			return JSONNull
		}
		return marshalValue(objValue.Elem(), info)
	default:
		log.Errorf("unsupport object %s %s", objValue.Type(), objValue.Interface())
		return JSONNull
	}
}
