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

package ovsdb

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"yunion.io/x/pkg/errors"
)

// NamedUuidPrefix marks a string as a reference to a row inserted in the same
// transaction.  It is the very same convention ovn-nbctl uses for its --id
// option, so row values can be shared between the two implementations
const NamedUuidPrefix = "@"

var uuidRegexp = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUuidString tells whether s has the shape of an ovsdb <uuid>
func IsUuidString(s string) bool {
	return uuidRegexp.MatchString(s)
}

// Uuid marshals to the RFC7047 <uuid> notation, ["uuid", "<36 chars>"]
type Uuid string

func (u Uuid) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]interface{}{"uuid", string(u)})
}

// NamedUuid marshals to the RFC7047 <named-uuid> notation, ["named-uuid", "<id>"]
type NamedUuid string

func (u NamedUuid) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]interface{}{"named-uuid", string(u)})
}

// UuidValue returns the <value> for a uuid typed column.  Strings prefixed
// with NamedUuidPrefix name a row inserted by the ongoing transaction,
// anything else is taken for a uuid of a row that already exists
func UuidValue(s string) interface{} {
	if strings.HasPrefix(s, NamedUuidPrefix) {
		return NamedUuid(s[len(NamedUuidPrefix):])
	}
	return Uuid(s)
}

// UuidValues is UuidValue for a list of references
func UuidValues(ss []string) []interface{} {
	r := make([]interface{}, len(ss))
	for i, s := range ss {
		r[i] = UuidValue(s)
	}
	return r
}

// RawUuid returns the <uuid> as it comes off the wire.  It is what the
// SetColumn methods of the generated row types expect to be fed with
func RawUuid(s string) interface{} {
	return []interface{}{"uuid", s}
}

// Set marshals to the RFC7047 <set> notation, ["set", [<value>, ...]].  The
// one element shorthand is never emitted, it is optional for senders
type Set []interface{}

func (s Set) MarshalJSON() ([]byte, error) {
	elems := []interface{}(s)
	if elems == nil {
		elems = []interface{}{}
	}
	return json.Marshal([2]interface{}{"set", elems})
}

// NewSet builds a <set> out of already encoded values
func NewSet(vals ...interface{}) Set {
	return Set(vals)
}

// MapEntry is one key value pair of a <map>
type MapEntry struct {
	Key   interface{}
	Value interface{}
}

// Map marshals to the RFC7047 <map> notation, ["map", [[<k>, <v>], ...]]
type Map []MapEntry

func (m Map) MarshalJSON() ([]byte, error) {
	pairs := make([][2]interface{}, len(m))
	for i, ent := range m {
		pairs[i] = [2]interface{}{ent.Key, ent.Value}
	}
	return json.Marshal([2]interface{}{"map", pairs})
}

// NewMapStringString builds a <map> of a Go map.  Keys are sorted so that the
// result is stable across calls
func NewMapStringString(m map[string]string) Map {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	r := make(Map, len(keys))
	for i, k := range keys {
		r[i] = MapEntry{Key: k, Value: m[k]}
	}
	return r
}

// Row is a RFC7047 <row>, a JSON object mapping column names to values
type Row map[string]interface{}

// Uuid returns the _uuid column of the row
func (row Row) Uuid() string {
	uuid, err := ParseUuid(row["_uuid"])
	if err != nil {
		return ""
	}
	return uuid
}

// Copy returns a shallow copy of the row.  Column values are treated as
// immutable throughout this package, so sharing them is fine
func (row Row) Copy() Row {
	r := make(Row, len(row))
	for k, v := range row {
		r[k] = v
	}
	return r
}

// parsePair decodes the ["<tag>", <v>] shape values of RFC7047 5.1 are
// wrapped in.  Values that do not have that shape are reported as untagged
func parsePair(val interface{}) (string, interface{}, bool) {
	arr, ok := val.([]interface{})
	if !ok || len(arr) != 2 {
		return "", nil, false
	}
	tag, ok := arr[0].(string)
	if !ok {
		return "", nil, false
	}
	return tag, arr[1], true
}

// ParseUuid extracts the uuid out of a ["uuid", <s>] or ["named-uuid", <s>] value
func ParseUuid(val interface{}) (string, error) {
	switch v := val.(type) {
	case Uuid:
		return string(v), nil
	case NamedUuid:
		return NamedUuidPrefix + string(v), nil
	}
	tag, v, ok := parsePair(val)
	if !ok {
		return "", errors.Wrapf(ErrValue, "uuid: unexpected value %#v", val)
	}
	s, ok := v.(string)
	if !ok {
		return "", errors.Wrapf(ErrValue, "uuid: not a string: %#v", v)
	}
	switch tag {
	case "uuid":
		return s, nil
	case "named-uuid":
		return NamedUuidPrefix + s, nil
	}
	return "", errors.Wrapf(ErrValue, "uuid: unexpected tag %q", tag)
}

// ParseSet extracts the elements of a <set>.  A bare value counts as a set of
// one element, as allowed by RFC7047 5.1
func ParseSet(val interface{}) ([]interface{}, error) {
	tag, v, ok := parsePair(val)
	if !ok {
		return []interface{}{val}, nil
	}
	if tag != "set" {
		// ["uuid", <s>] and the like are values of their own
		return []interface{}{val}, nil
	}
	elems, ok := v.([]interface{})
	if !ok {
		return nil, errors.Wrapf(ErrValue, "set: not an array: %#v", v)
	}
	return elems, nil
}

// ParseMapStringString extracts a <map> with string keys and string values
func ParseMapStringString(val interface{}) (map[string]string, error) {
	tag, v, ok := parsePair(val)
	if !ok || tag != "map" {
		return nil, errors.Wrapf(ErrValue, "map: unexpected value %#v", val)
	}
	pairs, ok := v.([]interface{})
	if !ok {
		return nil, errors.Wrapf(ErrValue, "map: not an array: %#v", v)
	}
	r := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		kv, ok := pair.([]interface{})
		if !ok || len(kv) != 2 {
			return nil, errors.Wrapf(ErrValue, "map: bad pair %#v", pair)
		}
		k, ok := kv[0].(string)
		if !ok {
			return nil, errors.Wrapf(ErrValue, "map: key not a string: %#v", kv[0])
		}
		v, ok := kv[1].(string)
		if !ok {
			return nil, errors.Wrapf(ErrValue, "map: value not a string: %#v", kv[1])
		}
		r[k] = v
	}
	return r, nil
}
