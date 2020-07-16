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

package sqlchemy

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/timeutils"
)

func getStringValue(dat interface{}) string {
	value := reflect.ValueOf(dat)
	switch value.Type() {
	case tristate.TriStateType:
		return dat.(tristate.TriState).String()
	case gotypes.TimeType:
		tm, ok := dat.(time.Time)
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
	}
	switch value.Kind() {
	case reflect.Bool:
		if value.Bool() {
			return "true"
		} else {
			return "false"
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", value.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", value.Float())
	case reflect.String:
		return value.String()
	}
	serializable, ok := value.Interface().(gotypes.ISerializable)
	if ok {
		return serializable.String()
	}
	log.Errorf("cannot convert %v to string", value)
	return ""
}

func setValueBySQLString(value reflect.Value, val string) error {
	if !value.CanSet() {
		return errors.Wrap(ErrReadOnly, "value is not settable")
	}
	switch value.Type() {
	case tristate.TriStateType:
		if val == "0" {
			value.Set(tristate.TriStateFalseValue)
		} else if val == "1" {
			value.Set(tristate.TriStateTrueValue)
		} else {
			value.Set(tristate.TriStateNoneValue)
		}
		return nil
	case gotypes.TimeType:
		if val != "0000-00-00 00:00:00" {
			tm, err := timeutils.ParseTimeStr(val)
			if err != nil {
				return errors.Wrap(err, "ParseTimeStr")
			}
			value.Set(reflect.ValueOf(tm))
		}
		return nil
	}
	switch value.Kind() {
	case reflect.Bool:
		if val == "0" {
			value.SetBool(false)
		} else {
			value.SetBool(true)
		}
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		valInt, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return errors.Wrap(err, "ParseInt")
		}
		value.SetInt(valInt)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		valUint, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return errors.Wrap(err, "ParseUint")
		}
		value.SetUint(valUint)
		return nil
	case reflect.Float32, reflect.Float64:
		valFloat, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return errors.Wrap(err, "ParseFloat")
		}
		value.SetFloat(valFloat)
		return nil
	case reflect.String:
		value.SetString(val)
		return nil
	case reflect.Slice:
		elemValue := reflect.New(value.Type().Elem()).Elem()
		err := setValueBySQLString(elemValue, val)
		if err != nil {
			return errors.Wrap(err, "reflect.Slice")
		}
		value.Set(reflect.Append(value, elemValue))
		return nil
	default:
		if valueType := value.Type(); valueType.Implements(gotypes.ISerializableType) {
			serializable, err := jsonutils.JSONDeserialize(valueType, val)
			if err != nil {
				return err
			}
			value.Set(reflect.ValueOf(serializable))
			return nil
		} else if value.Kind() == reflect.Ptr {
			if value.IsNil() {
				value.Set(reflect.New(value.Type().Elem()))
			}
			return setValueBySQLString(value.Elem(), val)
		} else {
			return errors.Wrapf(ErrNotSupported, "not supported type: %s", valueType)
		}
	}
}
