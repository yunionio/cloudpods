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

func marshalSlice(val reflect.Value) *JSONArray {
	objs := make([]JSONObject, val.Len())
	for i := 0; i < val.Len(); i += 1 {
		objs[i] = marshalValue(val.Index(i))
	}
	return NewArray(objs...)
}

func marshalMap(val reflect.Value) *JSONDict {
	keys := val.MapKeys()
	objPairs := make([]JSONPair, 0)
	for i := 0; i < len(keys); i += 1 {
		key := keys[i]
		val := marshalValue(val.MapIndex(key))
		if val != JSONNull {
			objPairs = append(objPairs, JSONPair{key: fmt.Sprintf("%s", key), val: val})
		}
	}
	return NewDict(objPairs...)
}

func marshalStruct(val reflect.Value) *JSONDict {
	objPairs := struct2JSONPairs(val)
	return NewDict(objPairs...)
}

func struct2JSONPairs(val reflect.Value) []JSONPair {
	structType := val.Type()
	objPairs := make([]JSONPair, 0)
	for i := 0; i < structType.NumField(); i += 1 {
		fieldType := structType.Field(i)
		if !gotypes.IsFieldExportable(fieldType.Name) { // unexportable field, ignore
			continue
		}
		if fieldType.Type.Kind() == reflect.Struct && fieldType.Anonymous { // embbed struct
			newPairs := struct2JSONPairs(val.Field(i))
			objPairs = append(objPairs, newPairs...)
		} else {
			key := reflectutils.GetStructFieldName(&fieldType) // utils.CamelSplit(fieldType.Name, "_")
			if key == "" {
				continue
			}
			val := marshalValue(val.Field(i))
			if val != JSONNull {
				objPair := JSONPair{key: key, val: val}
				objPairs = append(objPairs, objPair)
			}
		}
	}
	return objPairs
}

func marshalInt64(val int64) *JSONInt {
	return NewInt(val)
}

func marshalFloat64(val float64) *JSONFloat {
	return NewFloat(val)
}

func marshalBoolean(val bool) *JSONBool {
	if val {
		return JSONTrue
	} else {
		return JSONFalse
	}
}

func marshalTristate(val tristate.TriState) JSONObject {
	if val.IsTrue() {
		return JSONTrue
	} else if val.IsFalse() {
		return JSONFalse
	} else {
		return JSONNull
	}
}

func marshalString(val string) JSONObject {
	if len(val) == 0 {
		return JSONNull
	} else {
		return NewString(val)
	}
}

func marshalTime(val time.Time) *JSONString {
	if val.IsZero() {
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
	return marshalValue(objValue)
}

func marshalValue(objValue reflect.Value) JSONObject {
	switch objValue.Type() {
	case JSONDictPtrType, JSONArrayPtrType, JSONBoolPtrType, JSONIntPtrType, JSONFloatPtrType, JSONStringPtrType, JSONObjectType:
		if objValue.IsNil() {
			return JSONNull
		}
		return objValue.Interface().(JSONObject)
	case JSONDictType:
		json, ok := objValue.Interface().(JSONDict)
		if ok {
			return &json
		} else {
			return JSONNull
		}
	case JSONArrayType:
		json, ok := objValue.Interface().(JSONArray)
		if ok {
			return &json
		} else {
			return JSONNull
		}
	case JSONBoolType:
		json, ok := objValue.Interface().(JSONBool)
		if ok {
			return &json
		} else {
			return JSONNull
		}
	case JSONIntType:
		json, ok := objValue.Interface().(JSONInt)
		if ok {
			return &json
		} else {
			return JSONNull
		}
	case JSONFloatType:
		json, ok := objValue.Interface().(JSONFloat)
		if ok {
			return &json
		} else {
			return JSONNull
		}
	case JSONStringType:
		json, ok := objValue.Interface().(JSONString)
		if ok {
			return &json
		} else {
			return JSONNull
		}
	case tristate.TriStateType:
		tri, ok := objValue.Interface().(tristate.TriState)
		if ok {
			return marshalTristate(tri)
		} else {
			return JSONNull
		}
	}
	switch objValue.Kind() {
	case reflect.Slice, reflect.Array:
		return marshalSlice(objValue)
	case reflect.Struct:
		if objValue.Type() == gotypes.TimeType {
			return marshalTime(objValue.Interface().(time.Time))
		} else {
			return marshalStruct(objValue)
		}
	case reflect.Map:
		return marshalMap(objValue)
	case reflect.String:
		strValue := objValue.Convert(gotypes.StringType)
		return marshalString(strValue.Interface().(string))
	case reflect.Bool:
		return marshalBoolean(objValue.Interface().(bool))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		intValue := objValue.Convert(gotypes.Int64Type)
		return marshalInt64(intValue.Interface().(int64))
	case reflect.Float32, reflect.Float64:
		floatValue := objValue.Convert(gotypes.Float64Type)
		return marshalFloat64(floatValue.Interface().(float64))
	case reflect.Interface, reflect.Ptr:
		if objValue.IsNil() {
			return JSONNull
		}
		return marshalValue(objValue.Elem())
	default:
		log.Errorf("unsupport object %s %s", objValue.Type(), objValue.Interface())
		return JSONNull
	}
}
