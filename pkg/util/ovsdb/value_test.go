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
	"reflect"
	"testing"
)

func TestValueMarshal(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want string
	}{
		{
			name: "uuid",
			in:   Uuid("ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c"),
			want: `["uuid","ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c"]`,
		},
		{
			name: "named uuid",
			in:   NamedUuid("row0"),
			want: `["named-uuid","row0"]`,
		},
		{
			name: "empty set",
			in:   Set{},
			want: `["set",[]]`,
		},
		{
			name: "nil set",
			in:   Set(nil),
			want: `["set",[]]`,
		},
		{
			name: "string set",
			in:   NewSet("a", "b"),
			want: `["set",["a","b"]]`,
		},
		{
			name: "uuid set",
			in:   Set(UuidValues([]string{"@row0", "ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c"})),
			want: `["set",[["named-uuid","row0"],["uuid","ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c"]]]`,
		},
		{
			name: "empty map",
			in:   Map{},
			want: `["map",[]]`,
		},
		{
			name: "map is sorted by key",
			in:   NewMapStringString(map[string]string{"z": "0", "a": "1"}),
			want: `["map",[["a","1"],["z","0"]]]`,
		},
		{
			name: "condition",
			in:   Condition{Column: "_uuid", Function: FnEq, Value: Uuid("ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c")},
			want: `["_uuid","==",["uuid","ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c"]]`,
		},
		{
			name: "mutation",
			in:   Mutation{Column: "ports", Mutator: MutInsert, Value: Set(UuidValues([]string{"@row0"}))},
			want: `["ports","insert",["set",[["named-uuid","row0"]]]]`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := json.Marshal(c.in)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != c.want {
				t.Errorf("got  %s\nwant %s", string(got), c.want)
			}
		})
	}
}

func TestUuidValue(t *testing.T) {
	if got := UuidValue("@row0"); got != NamedUuid("row0") {
		t.Errorf("named uuid: got %#v", got)
	}
	if got := UuidValue("ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c"); got != Uuid("ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c") {
		t.Errorf("uuid: got %#v", got)
	}
}

func TestIsUuidString(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c", true},
		{"BA7D9B7E-B1F1-4A24-B3B6-2B0B8F1B6F4C", true},
		{"@row0", false},
		{"", false},
		{"ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4", false},
		{"ba7d9b7eb1f14a24b3b62b0b8f1b6f4c", false},
	}
	for _, c := range cases {
		if got := IsUuidString(c.in); got != c.want {
			t.Errorf("%q: got %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseValues(t *testing.T) {
	var (
		uuidVal = []interface{}{"uuid", "ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c"}
		namedV  = []interface{}{"named-uuid", "row0"}
		setVal  = []interface{}{"set", []interface{}{"a", "b"}}
		mapVal  = []interface{}{"map", []interface{}{
			[]interface{}{"k0", "v0"},
			[]interface{}{"k1", "v1"},
		}}
	)

	if got, err := ParseUuid(uuidVal); err != nil || got != "ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c" {
		t.Errorf("uuid: got %q, %v", got, err)
	}
	if got, err := ParseUuid(namedV); err != nil || got != "@row0" {
		t.Errorf("named uuid: got %q, %v", got, err)
	}
	if _, err := ParseUuid("nope"); err == nil {
		t.Errorf("uuid: want an error for a bare string")
	}

	if got, err := ParseSet(setVal); err != nil || !reflect.DeepEqual(got, []interface{}{"a", "b"}) {
		t.Errorf("set: got %#v, %v", got, err)
	}
	// rfc7047 5.1, a lone value stands for a set of one element
	if got, err := ParseSet("a"); err != nil || !reflect.DeepEqual(got, []interface{}{"a"}) {
		t.Errorf("set of one: got %#v, %v", got, err)
	}
	// a uuid is a value, not a set
	if got, err := ParseSet(uuidVal); err != nil || !reflect.DeepEqual(got, []interface{}{uuidVal}) {
		t.Errorf("uuid as set: got %#v, %v", got, err)
	}

	want := map[string]string{"k0": "v0", "k1": "v1"}
	if got, err := ParseMapStringString(mapVal); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("map: got %#v, %v", got, err)
	}
	if _, err := ParseMapStringString(setVal); err == nil {
		t.Errorf("map: want an error for a set")
	}
}

func TestRowUuid(t *testing.T) {
	row := Row{"_uuid": RawUuid("ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c")}
	if got := row.Uuid(); got != "ba7d9b7e-b1f1-4a24-b3b6-2b0b8f1b6f4c" {
		t.Errorf("got %q", got)
	}
	if got := (Row{}).Uuid(); got != "" {
		t.Errorf("no _uuid: got %q", got)
	}
}
