package reflectutils

import (
	"reflect"
	"strings"

	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"
)

type SStructFieldInfo struct {
	Ignore      bool
	OmitEmpty   bool
	OmitFalse   bool
	OmitZero    bool
	Name        string
	FieldName   string
	ForceString bool
	Tags        map[string]string
}

func ParseStructFieldJsonInfo(sf reflect.StructField) SStructFieldInfo {
	info := SStructFieldInfo{}
	info.FieldName = sf.Name
	info.OmitEmpty = true
	info.OmitZero = false
	info.OmitFalse = false

	info.Tags = utils.TagMap(sf.Tag)
	if val, ok := info.Tags["json"]; ok {
		keys := strings.Split(val, ",")
		if len(keys) > 0 {
			if keys[0] == "-" {
				if len(keys) > 1 {
					info.Name = keys[0]
				} else {
					info.Ignore = true
				}
			} else {
				info.Name = keys[0]
			}
		}
		if len(keys) > 1 {
			for _, k := range keys[1:] {
				switch strings.ToLower(k) {
				case "omitempty":
					info.OmitEmpty = true
				case "allowempty":
					info.OmitEmpty = false
				case "omitzero":
					info.OmitZero = true
				case "allowzero":
					info.OmitZero = false
				case "omitfalse":
					info.OmitFalse = true
				case "allowfalse":
					info.OmitFalse = false
				case "string":
					info.ForceString = true
				}
			}
		}
	}
	if val, ok := info.Tags["name"]; ok {
		info.Name = val
	}
	return info
}

func (info *SStructFieldInfo) MarshalName() string {
	if len(info.Name) > 0 {
		return info.Name
	}
	return utils.CamelSplit(info.FieldName, "_")
}

type SStructFieldValue struct {
	Info  SStructFieldInfo
	Value reflect.Value
}

type SStructFieldValueSet []SStructFieldValue

func FetchStructFieldValueSet(dataValue reflect.Value) SStructFieldValueSet {
	return fetchStructFieldValueSet(dataValue)
}

func fetchStructFieldValueSet(dataValue reflect.Value) SStructFieldValueSet {
	fields := SStructFieldValueSet{}
	dataType := dataValue.Type()
	for i := 0; i < dataType.NumField(); i += 1 {
		sf := dataType.Field(i)

		// ignore unexported field altogether
		if !gotypes.IsFieldExportable(sf.Name) {
			continue
		}
		fv := dataValue.Field(i)
		if !fv.IsValid() {
			continue
		}
		if sf.Anonymous {
			// T, *T
			switch fv.Kind() {
			case reflect.Ptr, reflect.Interface:
				if !fv.IsValid() || fv.IsNil() {
					continue
				}
				fv = fv.Elem()
			}
			// note that we regard anonymous interface field the
			// same as with anonymous struct field.  This is
			// different from how encoding/json handles struct
			// field of interface type.
			if fv.Kind() == reflect.Struct && sf.Type != gotypes.TimeType {
				subfields := fetchStructFieldValueSet(fv)
				fields = append(fields, subfields...)
				continue
			}
		}
		jsonInfo := ParseStructFieldJsonInfo(sf)
		fields = append(fields, SStructFieldValue{
			Info:  jsonInfo,
			Value: fv,
		})
	}
	return fields
}

func (set SStructFieldValueSet) GetStructFieldIndex(name string) int {
	for i := 0; i < len(set); i += 1 {
		jsonInfo := set[i].Info
		if jsonInfo.MarshalName() == name {
			return i
		}
		if utils.CamelSplit(jsonInfo.FieldName, "_") == utils.CamelSplit(name, "_") {
			return i
		}
		if jsonInfo.FieldName == name {
			return i
		}
		if jsonInfo.FieldName == utils.Capitalize(name) {
			return i
		}
	}
	return -1
}

func (set SStructFieldValueSet) GetValue(name string) (reflect.Value, bool) {
	idx := set.GetStructFieldIndex(name)
	if idx < 0 {
		return reflect.Value{}, false
	}
	return set[idx].Value, true
}

func (set SStructFieldValueSet) GetInterface(name string) (interface{}, bool) {
	idx := set.GetStructFieldIndex(name)
	if idx < 0 {
		return nil, false
	}
	if set[idx].Value.CanInterface() {
		return set[idx].Value.Interface(), true
	}
	return nil, false
}
