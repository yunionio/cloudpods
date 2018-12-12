package jsonutils

/**
jsonutils.Marshal

Convert any object to JSONObject

*/

import (
	"fmt"
	"reflect"
	"time"

	"strings"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
)

func marshalSlice(val reflect.Value, info *jsonMarshalInfo) JSONObject {
	if val.Len() == 0 && info != nil && info.omitEmpty {
		return JSONNull
	}
	objs := make([]JSONObject, val.Len())
	for i := 0; i < val.Len(); i += 1 {
		objs[i] = marshalValue(val.Index(i), nil)
	}
	arr := NewArray(objs...)
	if info != nil && info.forceString {
		return NewString(arr.String())
	} else {
		return arr
	}
}

func marshalMap(val reflect.Value, info *jsonMarshalInfo) JSONObject {
	keys := val.MapKeys()
	if len(keys) == 0 && info != nil && info.omitEmpty {
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
	if info != nil && info.forceString {
		return NewString(dict.String())
	} else {
		return dict
	}
}

func marshalStruct(val reflect.Value, info *jsonMarshalInfo) JSONObject {
	objPairs := struct2JSONPairs(val)
	if len(objPairs) == 0 && info != nil && info.omitEmpty {
		return JSONNull
	}
	dict := NewDict(objPairs...)
	if info != nil && info.forceString {
		return NewString(dict.String())
	} else {
		return dict
	}
}

type jsonMarshalInfo struct {
	ignore      bool
	omitEmpty   bool
	omitFalse   bool
	omitZero    bool
	name        string
	forceString bool
}

func parseJsonMarshalInfo(fieldTag reflect.StructTag) jsonMarshalInfo {
	info := jsonMarshalInfo{}
	info.omitEmpty = true
	info.omitZero = false
	info.omitFalse = false

	tags := utils.TagMap(fieldTag)
	if val, ok := tags["json"]; ok {
		keys := strings.Split(val, ",")
		if len(keys) > 0 {
			if keys[0] == "-" {
				if len(keys) > 1 {
					info.name = keys[0]
				} else {
					info.ignore = true
				}
			} else {
				info.name = keys[0]
			}
		}
		if len(keys) > 1 {
			for _, k := range keys[1:] {
				switch k {
				case "omitempty":
					info.omitEmpty = true
				case "allowempty":
					info.omitEmpty = false
				case "omitzero":
					info.omitZero = true
				case "allowzero":
					info.omitZero = false
				case "omitfalse":
					info.omitFalse = true
				case "allowfalse":
					info.omitFalse = false
				case "string":
					info.forceString = true
				}
			}
		}
	}
	if val, ok := tags["name"]; ok {
		info.name = val
	}
	return info
}

func struct2JSONPairs(val reflect.Value) []JSONPair {
	structType := val.Type()
	objPairs := make([]JSONPair, 0)
	for i := 0; i < structType.NumField(); i += 1 {
		sf := structType.Field(i)

		// ignore unexported field altogether
		if !gotypes.IsFieldExportable(sf.Name) {
			continue
		}

		if sf.Anonymous {
			fv := val.Field(i)

			// T, *T
			switch fv.Kind() {
			case reflect.Ptr, reflect.Interface:
				// ignore nil values completely
				if !fv.IsValid() || fv.IsNil() {
					continue
				}
				fv = fv.Elem()
			}
			// note that we regard anonymous interface field the
			// same as with anonymous struct field.  This is
			// different from how encoding/json handles struct
			// field of interface type.
			if fv.Kind() == reflect.Struct {
				newPairs := struct2JSONPairs(fv)
				objPairs = append(objPairs, newPairs...)
				continue
			}
		}

		jsonInfo := parseJsonMarshalInfo(sf.Tag)
		if jsonInfo.ignore {
			continue
		}
		key := jsonInfo.name
		if len(key) == 0 {
			key = utils.CamelSplit(sf.Name, "_")
		}
		val := marshalValue(val.Field(i), &jsonInfo)
		if val != nil && val != JSONNull {
			objPair := JSONPair{key: key, val: val}
			objPairs = append(objPairs, objPair)
		}
	}
	return objPairs
}

func marshalInt64(val int64, info *jsonMarshalInfo) JSONObject {
	if val == 0 && info != nil && info.omitZero {
		return JSONNull
	} else if info != nil && info.forceString {
		return NewString(fmt.Sprintf("%d", val))
	} else {
		return NewInt(val)
	}
}

func marshalFloat64(val float64, info *jsonMarshalInfo) JSONObject {
	if val == 0.0 && info != nil && info.omitZero {
		return JSONNull
	} else if info != nil && info.forceString {
		return NewString(fmt.Sprintf("%f", val))
	} else {
		return NewFloat(val)
	}
}

func marshalBoolean(val bool, info *jsonMarshalInfo) JSONObject {
	if !val && info != nil && info.omitFalse {
		return JSONNull
	} else if info != nil && info.forceString {
		return NewString(fmt.Sprintf("%v", val))
	} else {
		if val {
			return JSONTrue
		} else {
			return JSONFalse
		}
	}
}

func marshalTristate(val tristate.TriState, info *jsonMarshalInfo) JSONObject {
	if val.IsTrue() {
		return JSONTrue
	} else if val.IsFalse() {
		return JSONFalse
	} else {
		return JSONNull
	}
}

func marshalString(val string, info *jsonMarshalInfo) JSONObject {
	if len(val) == 0 && info != nil && info.omitEmpty {
		return JSONNull
	} else {
		return NewString(val)
	}
}

func marshalTime(val time.Time, info *jsonMarshalInfo) JSONObject {
	if val.IsZero() {
		if info != nil && info.omitEmpty {
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
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	return marshalValue(objValue, nil)
}

func marshalValue(objValue reflect.Value, info *jsonMarshalInfo) JSONObject {
	switch objValue.Type() {
	case JSONDictPtrType, JSONArrayPtrType, JSONBoolPtrType, JSONIntPtrType, JSONFloatPtrType, JSONStringPtrType, JSONObjectType:
		if objValue.IsNil() {
			return JSONNull
		}
		return objValue.Interface().(JSONObject)
	case JSONDictType:
		json, ok := objValue.Interface().(JSONDict)
		if ok {
			if len(json.data) == 0 && info != nil && info.omitEmpty {
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
			if len(json.data) == 0 && info != nil && info.omitEmpty {
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
			if !json.data && info != nil && info.omitEmpty {
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
			if json.data == 0 && info != nil && info.omitEmpty {
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
			if json.data == 0.0 && info != nil && info.omitEmpty {
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
			if len(json.data) == 0 && info != nil && info.omitEmpty {
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
	case reflect.Float32, reflect.Float64:
		floatValue := objValue.Convert(gotypes.Float64Type)
		return marshalFloat64(floatValue.Interface().(float64), info)
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
