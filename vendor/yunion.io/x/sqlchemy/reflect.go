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
	"database/sql"
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
		return fmt.Sprintf("%v", value.Float())
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

func ConvertValueToTime(val interface{}) time.Time {
	switch v := val.(type) {
	case string:
		tm, _ := timeutils.ParseTimeStr(v)
		return tm
	case time.Time:
		return v
	case int:
		return time.Unix(int64(v), 0)
	case int32:
		return time.Unix(int64(v), 0)
	case int64:
		return time.Unix(int64(v), 0)
	case uint:
		return time.Unix(int64(v), 0)
	case uint32:
		return time.Unix(int64(v), 0)
	case uint64:
		return time.Unix(int64(v), 0)
	case float32:
		return time.Unix(int64(v), int64((v-float32(int64(v)))*1000000000))
	case float64:
		return time.Unix(int64(v), int64((v-float64(int64(v)))*1000000000))
	case *string:
		tm, _ := timeutils.ParseTimeStr(*v)
		return tm
	case *time.Time:
		return *v
	case *int:
		return time.Unix(int64(*v), 0)
	case *int32:
		return time.Unix(int64(*v), 0)
	case *int64:
		return time.Unix(int64(*v), 0)
	case *uint:
		return time.Unix(int64(*v), 0)
	case *uint32:
		return time.Unix(int64(*v), 0)
	case *uint64:
		return time.Unix(int64(*v), 0)
	case *float32:
		return time.Unix(int64(*v), int64((*v-float32(int64(*v)))*1000000000))
	case *float64:
		return time.Unix(int64(*v), int64((*v-float64(int64(*v)))*1000000000))
	}
	return time.Time{}
}

func ConvertValueToInteger(val interface{}) int64 {
	switch v := val.(type) {
	case string:
		intv, _ := strconv.ParseInt(v, 10, 64)
		return intv
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	case time.Time:
		return v.Unix()
	case *string:
		intv, _ := strconv.ParseInt(*v, 10, 64)
		return intv
	case *int:
		return int64(*v)
	case *int32:
		return int64(*v)
	case *int64:
		return *v
	case *uint:
		return int64(*v)
	case *uint32:
		return int64(*v)
	case *uint64:
		return int64(*v)
	case *float32:
		return int64(*v)
	case *float64:
		return int64(*v)
	case *time.Time:
		return v.Unix()
	}
	return 0
}

func ConvertValueToFloat(val interface{}) float64 {
	switch v := val.(type) {
	case string:
		intv, _ := strconv.ParseFloat(v, 64)
		return intv
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
	case float64:
		return v
	case time.Time:
		return float64(v.Unix())
	case *string:
		intv, _ := strconv.ParseFloat(*v, 64)
		return intv
	case *int:
		return float64(*v)
	case *int32:
		return float64(*v)
	case *int64:
		return float64(*v)
	case *uint:
		return float64(*v)
	case *uint32:
		return float64(*v)
	case *uint64:
		return float64(*v)
	case *float32:
		return float64(*v)
	case *float64:
		return *v
	case *time.Time:
		return float64(v.Unix())
	}

	return 0
}

func ConvertValueToBool(val interface{}) bool {
	switch v := val.(type) {
	case string:
		v = strings.ToLower(v)
		return v == "1" || v == "true" || v == "ok" || v == "yes" || v == "on"
	case bool:
		return v
	case tristate.TriState:
		return v == tristate.True
	case int8:
		return v > 0
	case int16:
		return v > 0
	case int:
		return v > 0
	case int32:
		return v > 0
	case int64:
		return v > 0
	case uint8:
		return v > 0
	case uint16:
		return v > 0
	case uint:
		return v > 0
	case uint32:
		return v > 0
	case uint64:
		return v > 0
	case float32:
		return v > 0
	case float64:
		return v > 0
	case *string:
		if gotypes.IsNil(v) {
			return false
		}
		nv := strings.ToLower(*v)
		return nv == "1" || nv == "true" || nv == "ok" || nv == "yes" || nv == "on"
	case *bool:
		if gotypes.IsNil(v) {
			return false
		}
		return *v
	case *tristate.TriState:
		if gotypes.IsNil(v) {
			return false
		}
		return *v == tristate.True
	case *int8:
		if gotypes.IsNil(v) {
			return false
		}
		return *v > 0
	case *int16:
		if gotypes.IsNil(v) {
			return false
		}
		return *v > 0
	case *int:
		if gotypes.IsNil(v) {
			return false
		}
		return *v > 0
	case *int32:
		if gotypes.IsNil(v) {
			return false
		}
		return *v > 0
	case *int64:
		if gotypes.IsNil(v) {
			return false
		}
		return *v > 0
	case *uint8:
		if gotypes.IsNil(v) {
			return false
		}
		return *v > 0
	case *uint16:
		if gotypes.IsNil(v) {
			return false
		}
		return *v > 0
	case *uint:
		if gotypes.IsNil(v) {
			return false
		}
		return *v > 0
	case *uint32:
		if gotypes.IsNil(v) {
			return false
		}
		return *v > 0
	case *uint64:
		if gotypes.IsNil(v) {
			return false
		}
		return *v > 0
	case *float32:
		if gotypes.IsNil(v) {
			return false
		}
		return *v > 0
	case *float64:
		if gotypes.IsNil(v) {
			return false
		}
		return *v > 0
	}
	return false
}

func ConvertValueToTriState(val interface{}) tristate.TriState {
	switch v := val.(type) {
	case tristate.TriState:
		return v
	case string:
		switch strings.ToLower(v) {
		case "true", "yes", "on", "ok", "1":
			return tristate.True
		case "none", "null", "unknown":
			return tristate.None
		default:
			return tristate.False
		}
	case bool:
		if v {
			return tristate.True
		} else {
			return tristate.False
		}
	case sql.NullInt32:
		if v.Valid {
			return ConvertValueToTriState(v.Int32)
		}
		return tristate.None
	case int8:
		switch {
		case v > 0:
			return tristate.True
		default:
			return tristate.False
		}
	case int16:
		switch {
		case v > 0:
			return tristate.True
		default:
			return tristate.False
		}
	case int:
		switch {
		case v > 0:
			return tristate.True
		default:
			return tristate.False
		}
	case int32:
		switch {
		case v > 0:
			return tristate.True
		default:
			return tristate.False
		}
	case int64:
		switch {
		case v > 0:
			return tristate.True
		default:
			return tristate.False
		}
	case uint8:
		switch {
		case v > 0:
			return tristate.True
		default:
			return tristate.False
		}
	case uint16:
		switch {
		case v > 0:
			return tristate.True
		default:
			return tristate.False
		}
	case uint:
		switch {
		case v > 0:
			return tristate.True
		default:
			return tristate.False
		}
	case uint32:
		switch {
		case v > 0:
			return tristate.True
		default:
			return tristate.False
		}
	case uint64:
		switch {
		case v > 0:
			return tristate.True
		default:
			return tristate.False
		}
	case float32:
		switch {
		case v > 0:
			return tristate.True
		default:
			return tristate.False
		}
	case float64:
		switch {
		case v > 0:
			return tristate.True
		default:
			return tristate.False
		}
	case *string:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		switch strings.ToLower(*v) {
		case "true", "yes", "on", "ok", "1":
			return tristate.True
		case "none", "null", "unknown":
			return tristate.None
		default:
			return tristate.False
		}
	case *bool:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v {
			return tristate.True
		} else {
			return tristate.False
		}
	case *tristate.TriState:
		return *v
	case *int8:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v > 0 {
			return tristate.True
		} else {
			return tristate.False
		}
	case *int16:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v > 0 {
			return tristate.True
		} else {
			return tristate.False
		}
	case *int:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v > 0 {
			return tristate.True
		} else {
			return tristate.False
		}
	case *int32:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v > 0 {
			return tristate.True
		} else {
			return tristate.False
		}
	case *int64:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v > 0 {
			return tristate.True
		} else {
			return tristate.False
		}
	case *uint8:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v > 0 {
			return tristate.True
		} else {
			return tristate.False
		}
	case *uint16:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v > 0 {
			return tristate.True
		} else {
			return tristate.False
		}
	case *uint:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v > 0 {
			return tristate.True
		} else {
			return tristate.False
		}
	case *uint32:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v > 0 {
			return tristate.True
		} else {
			return tristate.False
		}
	case *uint64:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v > 0 {
			return tristate.True
		} else {
			return tristate.False
		}
	case *float32:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v > 0 {
			return tristate.True
		} else {
			return tristate.False
		}
	case *float64:
		if gotypes.IsNil(v) {
			return tristate.None
		}
		if *v > 0 {
			return tristate.True
		} else {
			return tristate.False
		}
	}
	return tristate.None
}

func ConvertValueToString(val interface{}) string {
	if gotypes.IsNil(val) {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case int8, int16, int, int32, int64, uint8, uint16, uint, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case bool:
		return fmt.Sprintf("%v", v)
	case time.Time:
		return timeutils.IsoTime(v)
	case *string:
		return *v
	case *int8:
		return fmt.Sprintf("%d", *v)
	case *int16:
		return fmt.Sprintf("%d", *v)
	case *int:
		return fmt.Sprintf("%d", *v)
	case *int32:
		return fmt.Sprintf("%d", *v)
	case *int64:
		return fmt.Sprintf("%d", *v)
	case *uint8:
		return fmt.Sprintf("%d", *v)
	case *uint16:
		return fmt.Sprintf("%d", *v)
	case *uint:
		return fmt.Sprintf("%d", *v)
	case *uint32:
		return fmt.Sprintf("%d", *v)
	case *uint64:
		return fmt.Sprintf("%d", *v)
	case *float32:
		return fmt.Sprintf("%f", *v)
	case *float64:
		return fmt.Sprintf("%f", *v)
	case *time.Time:
		return timeutils.IsoTime(*v)
	case tristate.TriState:
		return v.String()
	case *tristate.TriState:
		return (*v).String()
	}
	if reflect.ValueOf(val).Kind() == reflect.String {
		return val.(string)
	}
	return jsonutils.Marshal(val).String()
}
