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
//
// This struct has unexported fields initialized by exported functions in this
// package.  Do not construct a literal or modify the exported fields in an
// unmanaged way
//
// The tags of a field are read through Tag and TagMap.  The map behind them
// is shared with the other callers of the fetch functions, which is what
// makes reading a field several times cheap, so a caller may only modify what
// TagMap hands out, which is a copy of its own.
type SStructFieldInfo struct {
	// True if the field has json tag `json:"-"`
	Ignore bool

	// True if an empty string, slice, map, struct or dict is left out of
	// the json object.  A number and a boolean are not governed by this,
	// see OmitZero and OmitFalse.
	//
	// An empty value is left out by default, whether or not the json tag
	// asks for it; `allowempty` turns that off and `omitempty` states it.
	OmitEmpty bool

	// True if a false boolean is left out of the json object.  A false is
	// written by default; `omitfalse` turns that off and `allowfalse`
	// states it.
	OmitFalse bool

	// True if a zero number is left out of the json object.  A zero is
	// written by default; `omitzero` turns that off and `allowzero` states
	// it.
	OmitZero bool

	// Name can take the following values, in descreasing preference
	//
	//  1. value of "name" tag, e.g. `name:"a-name"`
	//  2. name of "json" tag, when it's not for ignoration
	//  3. kebab form of FieldName concatenated with "_" when Ignore is false
	//  4. empty string
	Name string

	// FieldName is the name of the go struct field
	FieldName string

	kebabFieldName string

	// True if the field has the "string" json tag option, which writes the
	// value as a json string
	ForceString bool

	// tags holds the tags of the field keyed by tag name, a tag without a
	// value being mapped to the empty string.  The map is shared with the
	// other callers of the fetch functions, so it has to be copied with
	// copyTags before being written to; read it through Tag or TagMap.
	tags map[string]string

	// aliases are the other names the field is looked up by, taken from
	// the "alias" tag.  Like tags it is shared, and has to be copied with
	// copyAliases before being written to.
	aliases []string
}

// copyTags takes a private copy of the tags, so that they can be written to
// without touching the ones this info was read out of.
func (s *SStructFieldInfo) copyTags() {
	tags := make(map[string]string, len(s.tags)+1)
	for k, v := range s.tags {
		tags[k] = v
	}
	s.tags = tags
}

// copyAliases takes a private copy of the aliases, so that they can be
// written to without touching the ones this info was read out of.
func (s *SStructFieldInfo) copyAliases() {
	aliases := make([]string, len(s.aliases))
	copy(aliases, s.aliases)
	s.aliases = aliases
}

func ParseStructFieldJsonInfo(sf reflect.StructField) SStructFieldInfo {
	return ParseFieldJsonInfo(sf.Name, sf.Tag)
}

func ParseFieldJsonInfo(name string, tag reflect.StructTag) SStructFieldInfo {
	info := SStructFieldInfo{}
	info.FieldName = name
	info.kebabFieldName = utils.CamelSplit(name, "_")
	info.OmitEmpty = true
	info.OmitZero = false
	info.OmitFalse = false

	info.tags = utils.TagMap(tag)
	if val, ok := info.tags["json"]; ok {
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
	if val, ok := info.tags["name"]; ok {
		info.Name = val
	}
	if !info.Ignore && len(info.Name) == 0 {
		info.Name = info.kebabFieldName
	}
	if val, ok := info.tags["alias"]; !info.Ignore && ok {
		info.aliases = strings.Split(val, ",")
	}
	return info
}

// MarshalName returns Name when it's not empty, otherwise it returns kebab
// form of the field name concatenated with "_"
func (info *SStructFieldInfo) MarshalName() string {
	if len(info.Name) > 0 {
		return info.Name
	}
	return info.kebabFieldName
}

// Tag returns the value of the tag named name and whether the field has it.
// A tag without a value is reported as present with an empty value.
func (info *SStructFieldInfo) Tag(name string) (string, bool) {
	val, ok := info.tags[name]
	return val, ok
}

// TagMap returns a copy of the tags of the field, which the caller owns and
// is free to modify.
func (info *SStructFieldInfo) TagMap() map[string]string {
	tags := make(map[string]string, len(info.tags))
	for k, v := range info.tags {
		tags[k] = v
	}
	return tags
}

type SStructFieldValue struct {
	Info  *SStructFieldInfo
	Value reflect.Value

	Parent *SEmbedStructFieldValue
}

// SEmbedStructFieldValue ties the fields enumerated for a nil embedded struct
// to the struct they belong to.  Field is the embedded pointer on the struct,
// Value is the value the fields were enumerated out of, and Parent is the
// entry of the enclosing embedded struct, if any.
type SEmbedStructFieldValue struct {
	Field reflect.Value
	Value reflect.Value

	Parent *SEmbedStructFieldValue
}

// adoptEmbeddedStruct assigns Value to Field for every nil embedded struct
// the field was enumerated through, which puts the enumerated fields back on
// the struct.  The assignments are made from the outermost embedded struct
// inwards, so that each one lands on a struct that is already part of the
// real one.  It reports whether the field is backed by the struct afterwards,
// which is what writing to it requires.
func (v *SStructFieldValue) adoptEmbeddedStruct() bool {
	chain := make([]*SEmbedStructFieldValue, 0, 2)
	for p := v.Parent; p != nil; p = p.Parent {
		chain = append(chain, p)
	}
	for i := len(chain) - 1; i >= 0; i -= 1 {
		p := chain[i]
		if !p.Field.IsValid() || p.Field.Kind() != reflect.Ptr || !p.Field.IsNil() {
			// nothing to adopt, the pointer is already there
			continue
		}
		if !p.Field.CanSet() {
			// the struct can not hold the embedded pointer
			return false
		}
		p.Field.Set(p.Value)
	}
	return true
}

type SStructFieldValueSet []SStructFieldValue

func FetchStructFieldValueSet(dataValue reflect.Value) SStructFieldValueSet {
	return expandAmbiguousPrefix(fetchStructFieldValueSet(dataValue, false))
}

func FetchStructFieldValueSetForWrite(dataValue reflect.Value) SStructFieldValueSet {
	return expandAmbiguousPrefix(fetchStructFieldValueSet(dataValue, true))
}

func FetchAllStructFieldValueSet(dataValue reflect.Value) SStructFieldValueSet {
	return expandAmbiguousPrefix(fetchStructFieldValueSet2(dataValue, false, nil, true))
}

func FetchAllStructFieldValueSetForWrite(dataValue reflect.Value) SStructFieldValueSet {
	return expandAmbiguousPrefix(fetchStructFieldValueSet2(dataValue, true, nil, true))
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

func fetchStructFieldValueSet(dataValue reflect.Value, allocatePtr bool) SStructFieldValueSet {
	return fetchStructFieldValueSet2(dataValue, allocatePtr, nil, false)
}

func fetchStructFieldValueSet2(dataValue reflect.Value, allocatePtr bool, tags map[string]string, includeIgnore bool) SStructFieldValueSet {
	return fetchStructFieldValueSet3(dataValue, allocatePtr, tags, includeIgnore, nil)
}

func fetchStructFieldValueSet3(dataValue reflect.Value, allocatePtr bool, tags map[string]string, includeIgnore bool, parent *SEmbedStructFieldValue) SStructFieldValueSet {
	if !dataValue.IsValid() || dataValue.Kind() != reflect.Struct {
		// A zero Value, a nil or a non struct value has no field to
		// enumerate.  Report it as such rather than through a panic.
		return SStructFieldValueSet{}
	}
	if allocatePtr && !dataValue.CanAddr() {
		// The value can not be modified in place, so a nil embedded
		// pointer can not be allocated on it.  The fields of such a
		// pointer are enumerated out of a value allocated on the side
		// instead, the same way as when the caller does not ask for
		// allocation; see the note below for how to write to them.
		allocatePtr = false
	}
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

		var efv *SEmbedStructFieldValue
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
					} else if fv.Kind() == reflect.Ptr && !allocatePtr {
						// The embedded pointer is nil, so the fields are
						// enumerated out of a value allocated on the side.
						// Value is a pointer to that value and the fields
						// below are the fields of the value it points at,
						// so assigning Value to Field puts them back on
						// the struct.  Callers writing to those fields
						// must do so first.
						efv = &SEmbedStructFieldValue{
							Field:  fv,
							Value:  reflect.New(fv.Type().Elem()),
							Parent: parent,
						}
						fv = efv.Value
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
				subfields := fetchStructFieldValueSet3(fv, allocatePtr, anonymousTags, includeIgnore, efv)
				fields = append(fields, subfields...)
				continue
			}
		}
		fieldInfo := fieldInfos[sf.Name]
		if !fieldInfo.Ignore || includeIgnore {
			structFieldVaule := SStructFieldValue{
				Info:  &fieldInfo,
				Value: fv,
			}
			if parent != nil {
				structFieldVaule.Parent = parent
			}
			fields = append(fields, structFieldVaule)
		}
	}
	if len(tags) > 0 {
		for i := range fields {
			fieldName := fields[i].Info.MarshalName()
			owned := false
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
				if !owned {
					// the tags may still be shared with other callers
					fields[i].Info.copyTags()
					owned = true
				}
				fields[i].Info.tags[k] = v
			}
		}
	}
	return fields
}

func (fields SStructFieldValueSet) GetStructFieldIndex(name string) int {
	indexes := fields.GetStructFieldIndexes(name)
	if len(indexes) > 0 {
		return indexes[0]
	}
	return -1
}

func (fields SStructFieldValueSet) GetStructFieldIndexes(name string) []int {
	return fields.GetStructFieldIndexes2(name, false)
}

func (fields SStructFieldValueSet) GetStructFieldIndexes2(name string, strictMode bool) []int {
	var (
		ret       []int
		kebabName string
		capName   string
	)
	if !strictMode && len(fields) > 0 {
		kebabName = utils.CamelSplit(name, "_")
		capName = utils.Capitalize(name)
	}
	for i := range fields {
		info := fields[i].Info
		if info.Ignore {
			continue
		}
		if info.MarshalName() == name {
			ret = append(ret, i)
			continue
		}
		if !strictMode {
			if info.kebabFieldName == kebabName {
				ret = append(ret, i)
			} else if info.FieldName == name {
				ret = append(ret, i)
			} else if info.FieldName == capName {
				ret = append(ret, i)
			} else if len(info.aliases) > 0 && utils.IsInArray(name, info.aliases) {
				ret = append(ret, i)
			}
		}
	}
	return ret
}

func (fields SStructFieldValueSet) GetStructFieldIndexesMap() map[string][]int {
	keyIndexMap := make(map[string][]int)
	for i := range fields {
		if fields[i].Info.Ignore {
			continue
		}
		key := fields[i].Info.MarshalName()
		values, ok := keyIndexMap[key]
		if !ok {
			values = make([]int, 0, 2)
		}
		keyIndexMap[key] = append(values, i)
	}
	return keyIndexMap
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
