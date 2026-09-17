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

package reflectutils

import (
	"fmt"
	"reflect"

	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
)

/*
func GetStructFieldName(field *reflect.StructField) string {
	tagMap := utils.TagMap(field.Tag)
	// var name string
	nameStr, _ := tagMap["name"]
	if len(nameStr) > 0 {
		return nameStr
	} else {
		jsonStr, _ := tagMap["json"]
		return toJsonKey(field.Name, jsonStr)
	}
}

func toJsonKey(fieldName, jsonTag string) string {
	jsonTag = strings.Replace(jsonTag, "omitempty", "", -1)
	words := utils.FindWords([]byte(jsonTag), 0)
	if len(words) == 0 {
		return utils.CamelSplit(fieldName, "_")
	}
	name := words[0]
	if name == "-" {
		return ""
	}
	return name
}

func FetchStructFieldNameValueInterfaces(dataValue reflect.Value) map[string]interface{} {
	fields := make(map[string]interface{})
	fetchStructFieldNameValueInterfaces(dataValue.Type(), dataValue, fields)
	return fields
}

func fetchStructFieldNameValueInterfaces(dataType reflect.Type, dataValue reflect.Value, fields map[string]interface{}) {
	for i := 0; i < dataType.NumField(); i += 1 {
		fieldType := dataType.Field(i)
		// log.Infof("%s %s %s", fieldType.Name, fieldType.Type, fieldType.Tag)
		if gotypes.IsFieldExportable(fieldType.Name) {
			fieldValue := dataValue.Field(i)
			if fieldType.Type.Kind() == reflect.Struct && fieldType.Anonymous {
				fetchStructFieldNameValueInterfaces(fieldType.Type, fieldValue, fields)
			} else if fieldValue.IsValid() && fieldValue.CanInterface() {
				val := fieldValue.Interface()
				// log.Debugf("val: %s %s %s", fieldType.Name, reflect.TypeOf(val), val, nil)
				if val != nil && !gotypes.IsNil(val) {
					name := GetStructFieldName(&fieldType)
					fields[name] = val
				}
			}
		}
	}
}

func FetchStructFieldNameValues(dataValue reflect.Value) map[string]reflect.Value {
	fields := make(map[string]reflect.Value)
	fetchStructFieldNameValues(dataValue.Type(), dataValue, fields)
	return fields
}

func fetchStructFieldNameValues(dataType reflect.Type, dataValue reflect.Value, fields map[string]reflect.Value) {
	for i := 0; i < dataType.NumField(); i += 1 {
		fieldType := dataType.Field(i)
		// log.Infof("%s %s %s", fieldType.Name, fieldType.Type, fieldType.Tag)
		if gotypes.IsFieldExportable(fieldType.Name) {
			fieldValue := dataValue.Field(i)
			if fieldType.Type.Kind() == reflect.Struct && fieldType.Anonymous {
				fetchStructFieldNameValues(fieldType.Type, fieldValue, fields)
			} else if fieldValue.IsValid() && fieldValue.CanSet() {
				name := GetStructFieldName(&fieldType)
				fields[name] = fieldValue
			}
		}
	}
}
*/

// FindStructFieldValue returns the field of dataValue named name.  The field
// has to be writable, so a field reached through a nil embedded pointer is
// only returned once that pointer has been put in place, which this does.
func FindStructFieldValue(dataValue reflect.Value, name string) (reflect.Value, bool) {
	set := FetchStructFieldValueSet(dataValue)
	idx := set.GetStructFieldIndex(name)
	if idx < 0 {
		return reflect.Value{}, false
	}
	if !set[idx].adoptEmbeddedStruct() || !set[idx].Value.CanSet() {
		return reflect.Value{}, false
	}
	return set[idx].Value, true
}

func FindStructFieldInterface(dataValue reflect.Value, name string) (interface{}, bool) {
	set := FetchStructFieldValueSet(dataValue)
	return set.GetInterface(name)
}

func FillEmbededStructValue(container reflect.Value, embed reflect.Value) bool {
	if !container.IsValid() || container.Kind() != reflect.Struct || !embed.IsValid() {
		return false
	}
	containerType := container.Type()
	embedType := embed.Type()
	for i := 0; i < containerType.NumField(); i += 1 {
		fieldType := containerType.Field(i)
		if fieldType.Type.Kind() != reflect.Struct || !fieldType.Anonymous {
			continue
		}
		fieldValue := container.Field(i)
		if !fieldValue.CanSet() {
			// an unexported embedded struct can not be assigned to, and
			// neither can anything inside it
			continue
		}
		if fieldType.Type == embedType {
			fieldValue.Set(embed)
			return true
		}
		if FillEmbededStructValue(fieldValue, embed) {
			return true
		}
	}
	return false
}

// SetStructFieldValue sets the field of structValue named fieldName to val.
// A field reached through a nil embedded pointer is written only once that
// pointer has been put in place, which this does.
func SetStructFieldValue(structValue reflect.Value, fieldName string, val reflect.Value) bool {
	set := FetchStructFieldValueSet(structValue)
	idx := set.GetStructFieldIndex(fieldName)
	if idx < 0 {
		return false
	}
	if !set[idx].adoptEmbeddedStruct() {
		return false
	}
	target := set[idx].Value
	if !target.CanSet() {
		return false
	}
	if !val.IsValid() || !val.Type().AssignableTo(target.Type()) {
		// report a failure instead of letting reflect.Value.Set panic
		return false
	}
	target.Set(val)
	return true
}

func ExpandInterface(val interface{}) []interface{} {
	value := reflect.Indirect(reflect.ValueOf(val))
	if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
		ret := make([]interface{}, value.Len())
		for i := 0; i < len(ret); i += 1 {
			ret[i] = value.Index(i).Interface()
		}
		return ret
	} else {
		return []interface{}{val}
	}
}

// tagetType must not be a pointer
func getAnonymouStructPointer(structValue reflect.Value, targetType reflect.Type) interface{} {
	structType := structValue.Type()
	if structType == targetType {
		if !structValue.CanInterface() {
			// the value was reached through an unexported field
			return nil
		}
		return structValue.Addr().Interface()
	}
	for i := 0; i < structValue.NumField(); i += 1 {
		fieldType := structType.Field(i)
		if !fieldType.Anonymous || !gotypes.IsFieldExportable(fieldType.Name) {
			// an unexported embedded struct can not be pointed at
			continue
		}
		fieldValue := structValue.Field(i)
		fieldT := fieldType.Type
		if fieldT.Kind() == reflect.Ptr {
			// an embedded pointer that is nil has nothing to point at
			if fieldValue.IsNil() {
				continue
			}
			fieldValue = fieldValue.Elem()
			fieldT = fieldT.Elem()
		}
		if fieldT.Kind() != reflect.Struct {
			continue
		}
		ptr := getAnonymouStructPointer(fieldValue, targetType)
		if ptr != nil {
			return ptr
		}
	}
	return nil
}

func FindAnonymouStructPointer(data interface{}, targetPtr interface{}) error {
	targetValue := reflect.ValueOf(targetPtr)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer to pointer")
	}
	targetValue = targetValue.Elem()
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer to pointer")
	}
	targetType := targetValue.Type().Elem()
	if targetType.Kind() != reflect.Struct {
		return fmt.Errorf("target type must be a struct")
	}
	structValue := reflect.ValueOf(data)
	if structValue.Kind() != reflect.Ptr {
		return fmt.Errorf("data type must be a pointer to struct")
	}
	structValue = reflect.ValueOf(data).Elem()
	if structValue.Kind() != reflect.Struct {
		return fmt.Errorf("data type must be a pointer to struct")
	}
	ptr := getAnonymouStructPointer(structValue, targetType)
	if ptr == nil {
		return fmt.Errorf("no anonymous struct found")
	}
	targetValue.Set(reflect.ValueOf(ptr))
	return nil
}

func StructContains(type1 reflect.Type, type2 reflect.Type) bool {
	if type1.Kind() != reflect.Struct || type2.Kind() != reflect.Struct {
		log.Errorf("types should be struct!")
		return false
	}
	if type1 == type2 {
		return true
	}
	for i := 0; i < type1.NumField(); i += 1 {
		field := type1.Field(i)
		if !field.Anonymous {
			continue
		}
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() != reflect.Struct {
			continue
		}
		if StructContains(fieldType, type2) {
			return true
		}
	}
	return false
}
