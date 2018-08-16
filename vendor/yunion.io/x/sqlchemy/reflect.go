package sqlchemy

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/timeutils"
)

func getStringValue(dat interface{}) string {
	value := reflect.ValueOf(dat)
	switch value.Type() {
	case gotypes.BoolType:
		if value.Bool() {
			return "true"
		} else {
			return "false"
		}
	case gotypes.IntType, gotypes.Int8Type, gotypes.Int16Type, gotypes.Int32Type, gotypes.Int64Type:
		return fmt.Sprintf("%d", value.Int())
	case gotypes.UintType, gotypes.Uint8Type, gotypes.Uint16Type, gotypes.Uint32Type, gotypes.Uint64Type:
		return fmt.Sprintf("%d", value.Uint())
	case gotypes.Float32Type, gotypes.Float64Type:
		return fmt.Sprintf("%f", value.Float())
	case gotypes.StringType:
		return value.String()
	case gotypes.TimeType:
		tm, ok := value.Interface().(time.Time)
		if !ok {
			log.Errorf("Fail to convert to time.Time %s", value)
		} else {
			return timeutils.MysqlTime(tm)
		}
	/*case jsonutils.JSONStringType, jsonutils.JSONIntType, jsonutils.JSONFloatType, jsonutils.JSONBoolType,
		jsonutils.JSONDictType, jsonutils.JSONArrayType:
	json, ok := value.Interface().(jsonutils.JSONObject)
	if !ok {
		log.Errorf("fail to convert to JSONObject", value)
	}else {
		return json.String()
	}*/
	case gotypes.Uint8SliceType:
		rawBytes, ok := value.Interface().([]byte)
		if !ok {
			log.Errorf("Fail to convert to bytes %s", value)
		} else {
			return string(rawBytes)
		}
	default:
		serializable, ok := value.Interface().(gotypes.ISerializable)
		if !ok {
			log.Errorf("cannot convert %v to string", value)
			return ""
		}
		return serializable.String()
	}
	return ""
}

func setValueBySQLString(value reflect.Value, val string) error {
	if !value.CanSet() {
		return fmt.Errorf("value is not settable")
	}
	switch value.Type() {
	case gotypes.BoolType:
		if val == "0" {
			value.SetBool(false)
		} else {
			value.SetBool(true)
		}
	case tristate.TriStateType:
		if val == "0" {
			value.Set(tristate.TriStateFalseValue)
		} else if val == "1" {
			value.Set(tristate.TriStateTrueValue)
		} else {
			value.Set(tristate.TriStateNoneValue)
		}
	case gotypes.IntType, gotypes.Int8Type, gotypes.Int16Type, gotypes.Int32Type, gotypes.Int64Type:
		valInt, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		value.SetInt(valInt)
	case gotypes.UintType, gotypes.Uint8Type, gotypes.Uint16Type, gotypes.Uint32Type, gotypes.Uint64Type:
		valUint, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return err
		}
		value.SetUint(valUint)
	case gotypes.Float32Type, gotypes.Float64Type:
		valFloat, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		value.SetFloat(valFloat)
	case gotypes.StringType:
		value.SetString(val)
	case gotypes.TimeType:
		if val != "0000-00-00 00:00:00" {
			tm, err := timeutils.ParseTimeStr(val)
			if err != nil {
				return err
			}
			value.Set(reflect.ValueOf(tm))
		}
	/*case jsonutils.JSONDictType, jsonutils.JSONArrayType, jsonutils.JSONStringType, jsonutils.JSONIntType,
		jsonutils.JSONFloatType, jsonutils.JSONBoolType, jsonutils.JSONObjectType:
	log.Debugf("Decode JSON value: $%s$", val)
	json, err := jsonutils.ParseString(val)
	if err != nil {
		return err
	}
	value.Set(reflect.ValueOf(json))*/
	case gotypes.BoolSliceType, gotypes.IntSliceType, gotypes.Int8SliceType, gotypes.Int16SliceType,
		gotypes.Int32SliceType, gotypes.Int64SliceType, gotypes.UintSliceType, gotypes.Uint8SliceType,
		gotypes.Uint16SliceType, gotypes.Uint32SliceType, gotypes.Uint64SliceType,
		gotypes.Float32SliceType, gotypes.Float64SliceType, gotypes.StringSliceType:
		reflect.Append(value, reflect.ValueOf(val))
	default:
		valueType := value.Type()
		if !valueType.Implements(gotypes.ISerializableType) {
			return fmt.Errorf("not supported type: %s", valueType)
		}
		serializable, err := jsonutils.JSONDeserialize(valueType, val)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(serializable))
	}
	return nil
}
