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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/timeutils"
)

func getQuoteStringValue(dat interface{}) string {
	value := reflect.ValueOf(dat)
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", value.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", value.Float())
	}
	strVal := GetStringValue(dat)
	strVal = strings.ReplaceAll(strVal, "'", "\\'")
	return "'" + strVal + "'"
}

func GetStringValue(dat interface{}) string {
	switch g := dat.(type) {
	case tristate.TriState:
		return g.String()
	case time.Time:
		return timeutils.MysqlTime(g)
	case []byte:
		return string(g)
	}
	value := reflect.Indirect(reflect.ValueOf(dat))
	switch value.Kind() {
	case reflect.Bool:
		if value.Bool() {
			return "true"
		}
		return "false"
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
	} else {
		return jsonutils.Marshal(value.Interface()).String()
	}
}

func setValueBySQLString(value reflect.Value, val string) error {
	if !value.CanSet() {
		return errors.Wrap(ErrReadOnly, "value is not settable")
	}
	switch value.Type() {
	case tristate.TriStateType:
		if val == "1" {
			value.Set(tristate.TriStateTrueValue)
		} else if val == "0" {
			value.Set(tristate.TriStateFalseValue)
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
		jsonV, err := jsonutils.ParseString(val)
		if err != nil {
			return errors.Wrapf(err, "jsonutils.ParseString %s", val)
		}
		if jsonV == jsonutils.JSONNull {
			return nil
		}
		jsonA, err := jsonV.GetArray()
		if err != nil {
			return errors.Wrap(err, "jsonV.GetArray")
		}
		sliceValue := reflect.MakeSlice(value.Type(), 0, len(jsonA))
		value.Set(sliceValue)
		for i := range jsonA {
			elemValue := reflect.New(value.Type().Elem()).Elem()
			jsonStr, _ := jsonA[i].GetString()
			err := setValueBySQLString(elemValue, jsonStr)
			if err != nil {
				return errors.Wrapf(err, "TestSetValueBySQLString %s", jsonA[i].String())
			}
			value.Set(reflect.Append(value, elemValue))
		}
		return nil
	case reflect.Map:
		jsonV, err := jsonutils.ParseString(val)
		if err != nil {
			return errors.Wrapf(err, "jsonutils.ParseString %s", val)
		}
		if jsonV == jsonutils.JSONNull {
			return nil
		}
		mapValue := reflect.MakeMap(value.Type())
		value.Set(mapValue)
		err = jsonV.Unmarshal(mapValue.Interface())
		if err != nil {
			return errors.Wrapf(err, "jsonV.Unmarshal")
		}
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
			jsonV, err := jsonutils.ParseString(val)
			if err != nil {
				return errors.Wrapf(err, "%s not a json string: %s", val, err)
			}
			if jsonV == jsonutils.JSONNull {
				return nil
			}
			newVal := reflect.New(value.Type())
			err = jsonV.Unmarshal(newVal.Interface())
			if err != nil {
				return errors.Wrap(err, "Unmarshal fail")
			}
			value.Set(reflect.Indirect(newVal))
			return nil
		}
	}
}
