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

func FindStructFieldValue(dataValue reflect.Value, name string) (reflect.Value, bool) {
	set := FetchStructFieldValueSet(dataValue)
	val, find := set.GetValue(name)
	if find && val.CanSet() {
		return val, true
	}
	return reflect.Value{}, false
}

func FindStructFieldInterface(dataValue reflect.Value, name string) (interface{}, bool) {
	set := FetchStructFieldValueSet(dataValue)
	return set.GetInterface(name)
}

func FillEmbededStructValue(container reflect.Value, embed reflect.Value) bool {
	containerType := container.Type()
	for i := 0; i < containerType.NumField(); i += 1 {
		fieldType := containerType.Field(i)
		fieldValue := container.Field(i)
		if fieldType.Type.Kind() == reflect.Struct && fieldType.Anonymous {
			if fieldType.Type == embed.Type() {
				fieldValue.Set(embed)
				return true
			} else {
				filled := FillEmbededStructValue(fieldValue, embed)
				if filled {
					return true
				}
			}
		}

	}
	return false
}

func SetStructFieldValue(structValue reflect.Value, fieldName string, val reflect.Value) bool {
	set := FetchStructFieldValueSet(structValue)
	target, find := set.GetValue(fieldName)
	if !find {
		return false
	}
	if !target.CanSet() {
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
		return structValue.Addr().Interface()
	}
	for i := 0; i < structValue.NumField(); i += 1 {
		fieldType := structType.Field(i)
		if fieldType.Anonymous && fieldType.Type.Kind() == reflect.Struct {
			ptr := getAnonymouStructPointer(structValue.Field(i), targetType)
			if ptr != nil {
				return ptr
			}
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
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			contains := StructContains(field.Type, type2)
			if contains {
				return true
			}
		}
	}
	return false
}
