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

package gotypes

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
)

const (
	EMPTYSTR = ""
)

var (
	boolValue         bool
	intValue          int
	int8Value         int8
	int16Value        int16
	int32Value        int32
	int64Value        int64
	uintValue         uint
	uint8Value        uint8
	uint16Value       uint16
	uint32Value       uint32
	uint64Value       uint64
	float32Value      float32
	float64Value      float64
	stringValue       string
	boolSliceValue    []bool
	intSliceValue     []int
	int8SliceValue    []int8
	int16SliceValue   []int16
	int32SliceValue   []int32
	int64SliceValue   []int64
	uintSliceValue    []uint
	uint8SliceValue   []uint8
	uint16SliceValue  []uint16
	uint32SliceValue  []uint32
	uint64SliceValue  []uint64
	float32SliceValue []float32
	float64SliceValue []float64
	stringSliceValue  []string
	timeValue         time.Time
)

var (
	BoolType         = reflect.TypeOf(boolValue)
	IntType          = reflect.TypeOf(intValue)
	Int8Type         = reflect.TypeOf(int8Value)
	Int16Type        = reflect.TypeOf(int16Value)
	Int32Type        = reflect.TypeOf(int32Value)
	Int64Type        = reflect.TypeOf(int64Value)
	UintType         = reflect.TypeOf(uintValue)
	Uint8Type        = reflect.TypeOf(uint8Value)
	Uint16Type       = reflect.TypeOf(uint16Value)
	Uint32Type       = reflect.TypeOf(uint32Value)
	Uint64Type       = reflect.TypeOf(uint64Value)
	Float32Type      = reflect.TypeOf(float32Value)
	Float64Type      = reflect.TypeOf(float64Value)
	StringType       = reflect.TypeOf(stringValue)
	BoolSliceType    = reflect.TypeOf(boolSliceValue)
	IntSliceType     = reflect.TypeOf(intSliceValue)
	Int8SliceType    = reflect.TypeOf(int8SliceValue)
	Int16SliceType   = reflect.TypeOf(int16SliceValue)
	Int32SliceType   = reflect.TypeOf(int32SliceValue)
	Int64SliceType   = reflect.TypeOf(int64SliceValue)
	UintSliceType    = reflect.TypeOf(uintSliceValue)
	Uint8SliceType   = reflect.TypeOf(uint8SliceValue)
	Uint16SliceType  = reflect.TypeOf(uint16SliceValue)
	Uint32SliceType  = reflect.TypeOf(uint32SliceValue)
	Uint64SliceType  = reflect.TypeOf(uint64SliceValue)
	Float32SliceType = reflect.TypeOf(float32SliceValue)
	Float64SliceType = reflect.TypeOf(float64SliceValue)
	StringSliceType  = reflect.TypeOf(stringSliceValue)
	TimeType         = reflect.TypeOf(timeValue)
)

func ParseValue(val string, tp reflect.Type) (reflect.Value, error) {
	switch tp.Kind() {
	case reflect.Bool:
		val_bool, err := strconv.ParseBool(val)
		return reflect.ValueOf(val_bool), err
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val_int, err := strconv.ParseInt(val, 10, 64)
		switch tp.Kind() {
		case reflect.Int:
			return reflect.ValueOf(int(val_int)), err
		case reflect.Int8:
			return reflect.ValueOf(int8(val_int)), err
		case reflect.Int16:
			return reflect.ValueOf(int16(val_int)), err
		case reflect.Int32:
			return reflect.ValueOf(int32(val_int)), err
		default:
			return reflect.ValueOf(val_int), err
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val_uint, err := strconv.ParseUint(val, 10, 64)
		switch tp.Kind() {
		case reflect.Uint:
			return reflect.ValueOf(uint(val_uint)), err
		case reflect.Uint8:
			return reflect.ValueOf(uint8(val_uint)), err
		case reflect.Uint16:
			return reflect.ValueOf(uint16(val_uint)), err
		case reflect.Uint32:
			return reflect.ValueOf(uint32(val_uint)), err
		default:
			return reflect.ValueOf(val_uint), err
		}
	case reflect.Float32, reflect.Float64:
		val_float, err := strconv.ParseFloat(val, 64)
		if tp.Kind() == reflect.Float32 {
			return reflect.ValueOf(float32(val_float)), err
		} else {
			return reflect.ValueOf(val_float), err
		}
	case reflect.String:
		v := reflect.New(tp).Elem()
		v.SetString(val)
		return v, nil
	case reflect.Ptr:
		tpElem := tp.Elem()
		rv, err := ParseValue(val, tpElem)
		if err != nil {
			return reflect.ValueOf(val), fmt.Errorf("Cannot parse %s to %s", val, tp)
		}
		rvv := reflect.New(tpElem)
		rvv.Elem().Set(rv)
		return rvv, nil
	case reflect.Slice, reflect.Array:
		values := utils.FindWords([]byte(val), 0)
		sliceVal := reflect.MakeSlice(reflect.SliceOf(tp.Elem()), len(values), len(values))
		for i, vv := range values {
			vvv, err := ParseValue(vv, tp.Elem())
			if err != nil {
				return sliceVal, fmt.Errorf("Cannot parse %s to %s", vv, tp.Elem())
			}
			sliceVal.Index(i).Set(vvv)
		}
		return sliceVal, nil
	default:
		if tp == TimeType {
			tm, e := timeutils.ParseTimeStr(val)
			return reflect.ValueOf(tm), e
		} else {
			return reflect.ValueOf(val), fmt.Errorf("Cannot parse %s to %s", val, tp)
		}
	}
}

func SetValue(value reflect.Value, valStr string) error {
	if !value.CanSet() {
		return fmt.Errorf("Value is not settable")
	}
	parseValue, err := ParseValue(valStr, value.Type())
	if err != nil {
		return err
	}
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		value.Set(reflect.AppendSlice(value, parseValue))
	default:
		value.Set(parseValue)
	}
	return nil
}

func AppendValues(value reflect.Value, vals ...string) error {
	var e error = nil
	for _, val := range vals {
		e = AppendValue(value, val)
		if e != nil {
			break
		}
	}
	return e
}

func SliceBaseType(tp reflect.Type) reflect.Type {
	switch tp {
	case BoolSliceType:
		return BoolType
	case IntSliceType:
		return IntType
	case Int8SliceType:
		return Int8Type
	case Int16SliceType:
		return Int16Type
	case Int32SliceType:
		return Int32Type
	case Int64SliceType:
		return Int64Type
	case UintSliceType:
		return UintType
	case Uint8SliceType:
		return Uint8Type
	case Uint16SliceType:
		return Uint16Type
	case Uint32SliceType:
		return Uint32Type
	case Uint64SliceType:
		return Uint64Type
	case Float32SliceType:
		return Float32Type
	case Float64SliceType:
		return Float64Type
	case StringSliceType:
		return StringType
	default:
		return nil
	}
}

func AppendValue(value reflect.Value, val string) error {
	if value.Kind() != reflect.Slice {
		return fmt.Errorf("Cannot append to non-slice type")
	}
	tp := value.Type().Elem()
	val_raw, e := ParseValue(val, tp)
	if e != nil {
		return e
	}
	value.Set(reflect.Append(value, val_raw))
	return nil
}

func InCollection(obj interface{}, array interface{}) bool {
	var arrVal = reflect.ValueOf(array)
	var arrKind = arrVal.Type().Kind()
	var arrSet []reflect.Value
	if arrKind == reflect.Map {
		arrSet = arrVal.MapKeys()
	} else if arrKind == reflect.Array || arrKind == reflect.Slice {
		arrSet = make([]reflect.Value, 0)
		for i := 0; i < arrVal.Len(); i++ {
			arrSet = append(arrSet, arrVal.Index(i))
		}
	} else {
		return false
	}
	for _, arrObj := range arrSet {
		if reflect.DeepEqual(obj, arrObj.Interface()) {
			return true
		}
	}
	return false
}

func IsFieldExportable(fieldName string) bool {
	return strings.ToUpper(fieldName[:1]) == fieldName[:1]
}

func IsNil(val interface{}) bool {
	if val == nil {
		return true
	}
	valValue := reflect.ValueOf(val)
	switch valValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Interface:
		return valValue.IsNil()
	default:
		return val == nil
	}
}

func GetInstanceTypeName(instance interface{}) string {
	val := reflect.Indirect(reflect.ValueOf(instance))
	typeStr := val.Type().String()
	dotPos := strings.IndexByte(typeStr, '.')
	if dotPos >= 0 {
		return typeStr[dotPos+1:]
	} else {
		return typeStr
	}
}

// Convert vals to slice of pointer val's element type
//
// This is for the following use cases, not for general type conversion
//
//  - Convert []Interface to []ConcreteType
//  - Convert []ConcreateType to []Interface
func ConvertSliceElemType(vals interface{}, val interface{}) interface{} {
	origVals := reflect.ValueOf(vals)
	origType := origVals.Type()
	if origKind := origType.Kind(); origKind != reflect.Slice && origKind != reflect.Array {
		panic(fmt.Sprintf("only accepts slice or array, got %s", origKind))
	}
	var sliceType reflect.Type
	if val == nil {
		if origVals.Len() == 0 {
			// nothing to do
			return vals
		}
		val = origVals.Index(0).Interface()
		sliceType = reflect.SliceOf(reflect.TypeOf(val))
	} else {
		valType := reflect.TypeOf(val)
		if valKind := valType.Kind(); valKind != reflect.Ptr {
			panic(fmt.Sprintf("val when not nil must be ptr kind, got %s", valKind))
		}
		sliceType = reflect.SliceOf(valType.Elem())
	}
	length := origVals.Len()
	r := reflect.MakeSlice(sliceType, length, length)
	for i := 0; i < length; i++ {
		origVal := origVals.Index(i)
		if origVal.Kind() == reflect.Interface {
			origVal = origVal.Elem()
		}
		r.Index(i).Set(origVal)
	}
	return r.Interface()
}
