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
	"reflect"
	"strings"
	"sync"

	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/utils"
)

// SStructFieldInfo describes struct field, especially behavior for (json)
// marshal
type SStructFieldInfo struct {
	// True if the field has json tag `json:"-"`
	Ignore    bool
	OmitEmpty bool
	OmitFalse bool
	OmitZero  bool

	// Name can take the following values, in descreasing preference
	//
	//  1. value of "name" tag, e.g. `name:"a-name"`
	//  2. name of "json" tag, when it's not for ignoration
	//  3. kebab form of FieldName concatenated with "_" when Ignore is false
	//  4. empty string
	Name        string
	FieldName   string
	ForceString bool
	Tags        map[string]string
}

func (s *SStructFieldInfo) updateTags(k, v string) {
	s.Tags[k] = v
}

func (s SStructFieldInfo) deepCopy() *SStructFieldInfo {
	scopy := SStructFieldInfo{
		Ignore:      s.Ignore,
		OmitEmpty:   s.OmitEmpty,
		OmitFalse:   s.OmitFalse,
		OmitZero:    s.OmitZero,
		Name:        s.Name,
		FieldName:   s.FieldName,
		ForceString: s.ForceString,
	}
	tags := make(map[string]string, len(s.Tags))
	for k, v := range s.Tags {
		tags[k] = v
	}
	scopy.Tags = tags
	return &scopy
}

func ParseStructFieldJsonInfo(sf reflect.StructField) SStructFieldInfo {
	return ParseFieldJsonInfo(sf.Name, sf.Tag)
}

func ParseFieldJsonInfo(name string, tag reflect.StructTag) SStructFieldInfo {
	info := SStructFieldInfo{}
	info.FieldName = name
	info.OmitEmpty = true
	info.OmitZero = false
	info.OmitFalse = false

	info.Tags = utils.TagMap(tag)
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
	if !info.Ignore && len(info.Name) == 0 {
		info.Name = utils.CamelSplit(info.FieldName, "_")
	}
	return info
}

// MarshalName returns Name when it's not empty, otherwise it returns kebab
// form of the field name concatenated with "_"
func (info *SStructFieldInfo) MarshalName() string {
	if len(info.Name) > 0 {
		return info.Name
	}
	return utils.CamelSplit(info.FieldName, "_")
}

type SStructFieldValue struct {
	Info  *SStructFieldInfo
	Value reflect.Value
}

type SStructFieldValueSet []SStructFieldValue

func FetchStructFieldValueSet(dataValue reflect.Value) SStructFieldValueSet {
	return fetchStructFieldValueSet(dataValue, false, nil)
}

func FetchStructFieldValueSetForWrite(dataValue reflect.Value) SStructFieldValueSet {
	return fetchStructFieldValueSet(dataValue, true, nil)
}

type sStructFieldInfoMap map[string]SStructFieldInfo

func newStructFieldInfoMap(caps int) sStructFieldInfoMap {
	return make(map[string]SStructFieldInfo, caps)
}

var structFieldInfoCache sync.Map

func fetchCacheStructFieldInfos(dataType reflect.Type) sStructFieldInfoMap {
	if r, ok := structFieldInfoCache.Load(dataType); ok {
		return r.(sStructFieldInfoMap)
	}
	infos := fetchStructFieldInfos(dataType)
	structFieldInfoCache.Store(dataType, infos)
	return infos
}

func fetchStructFieldInfos(dataType reflect.Type) sStructFieldInfoMap {
	smap := newStructFieldInfoMap(dataType.NumField())
	for i := 0; i < dataType.NumField(); i += 1 {
		sf := dataType.Field(i)
		if !gotypes.IsFieldExportable(sf.Name) {
			continue
		}
		if sf.Anonymous {
			// call ParseStructFieldJsonInfo for sf if sft.Kind() is reflect.Interface:
			// if the corresponding value is reflect.Struct, this item in fieldInfos will be ignored,
			// otherwise this item in fieldInfos will be used correctly
			sft := sf.Type
			if sft.Kind() == reflect.Ptr {
				sft = sft.Elem()
			}
			if sft.Kind() == reflect.Struct && sft != gotypes.TimeType {
				continue
			}
		}
		smap[sf.Name] = ParseStructFieldJsonInfo(sf)
	}
	return smap
}

func fetchStructFieldValueSet(dataValue reflect.Value, allocatePtr bool, tags map[string]string) SStructFieldValueSet {
	fields := SStructFieldValueSet{}
	dataType := dataValue.Type()
	fieldInfos := fetchCacheStructFieldInfos(dataType)
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
				if !fv.IsValid() {
					continue
				}
				if fv.IsNil() {
					if fv.Kind() == reflect.Ptr && allocatePtr {
						fv.Set(reflect.New(fv.Type().Elem()))
					} else {
						continue
					}
				}
				fv = fv.Elem()
			}
			// note that we regard anonymous interface field the
			// same as with anonymous struct field.  This is
			// different from how encoding/json handles struct
			// field of interface type.
			if fv.Kind() == reflect.Struct && sf.Type != gotypes.TimeType {
				anonymousTags := utils.TagMap(sf.Tag)
				subfields := fetchStructFieldValueSet(fv, allocatePtr, anonymousTags)
				fields = append(fields, subfields...)
				continue
			}
		}
		fieldInfo := fieldInfos[sf.Name].deepCopy()
		if !fieldInfo.Ignore {
			fields = append(fields, SStructFieldValue{
				Info:  fieldInfo,
				Value: fv,
			})
		}
	}
	if len(tags) > 0 {
		for i := range fields {
			fieldName := fields[i].Info.MarshalName()
			for k, v := range tags {
				target := ""
				pos := strings.Index(k, "->")
				if pos > 0 {
					target = k[:pos]
					k = k[pos+2:]
				}
				if len(target) > 0 && target != fieldName {
					continue
				}
				fields[i].Info.updateTags(k, v)
			}
		}
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

func (set SStructFieldValueSet) GetStructFieldIndexes(name string) []int {
	ret := make([]int, 0)
	for i := 0; i < len(set); i += 1 {
		jsonInfo := set[i].Info
		if jsonInfo.MarshalName() == name {
			ret = append(ret, i)
			continue
		}
		if utils.CamelSplit(jsonInfo.FieldName, "_") == utils.CamelSplit(name, "_") {
			ret = append(ret, i)
			continue
		}
		if jsonInfo.FieldName == name {
			ret = append(ret, i)
			continue
		}
		if jsonInfo.FieldName == utils.Capitalize(name) {
			ret = append(ret, i)
			continue
		}
	}
	return ret
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
