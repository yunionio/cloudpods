package reflectutils

import (
	"reflect"

	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"
)

func GetStructFieldName(field *reflect.StructField) string {
	tagMap := utils.TagMap(field.Tag)
	// var name string
	nameStr, _ := tagMap["name"]
	if len(nameStr) > 0 {
		return nameStr
	} else {
		jsonStr, _ := tagMap["json"]
		if len(jsonStr) > 0 {
			return jsonStr
		} else {
			return utils.CamelSplit(field.Name, "_")
		}
	}
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

func FindStructFieldValue(dataValue reflect.Value, name string) (reflect.Value, bool) {
	dataType := dataValue.Type()
	for i := 0; i < dataType.NumField(); i += 1 {
		fieldType := dataType.Field(i)
		if gotypes.IsFieldExportable(fieldType.Name) {
			fieldValue := dataValue.Field(i)
			if fieldType.Type.Kind() == reflect.Struct && fieldType.Type != gotypes.TimeType {
				val, find := FindStructFieldValue(fieldValue, name)
				if find {
					return val, find
				}
			} else if fieldValue.CanSet() {
				fName := GetStructFieldName(&fieldType)
				if fName == name {
					return fieldValue, true
				}
			}
		}
	}
	return reflect.Value{}, false
}

func FindStructFieldInterface(dataValue reflect.Value, name string) (interface{}, bool) {
	dataType := dataValue.Type()
	for i := 0; i < dataType.NumField(); i += 1 {
		fieldType := dataType.Field(i)
		if gotypes.IsFieldExportable(fieldType.Name) {
			fieldValue := dataValue.Field(i)
			if fieldType.Type.Kind() == reflect.Struct && fieldType.Type != gotypes.TimeType {
				val, find := FindStructFieldInterface(fieldValue, name)
				if find {
					return val, find
				}
			} else if fieldValue.CanInterface() {
				fName := GetStructFieldName(&fieldType)
				if fName == name {
					return fieldValue.Interface(), true
				}
			}
		}
	}
	return nil, false
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
	dataType := structValue.Type()
	for i := 0; i < dataType.NumField(); i += 1 {
		fieldType := dataType.Field(i)
		if gotypes.IsFieldExportable(fieldType.Name) {
			fName := GetStructFieldName(&fieldType)
			if fName == fieldName {
				fieldValue := structValue.Field(i)
				fieldValue.Set(val)
				return true
			}
		}
	}
	return false
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
