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

package options

import (
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"
)

type S struct {
	In   interface{}
	Want string
}

func testS(t *testing.T, c *S) {
	jsonGot, err := StructToParams(c.In)
	if err != nil {
		t.Errorf("StructToParams failed: in: %#v: err: %s",
			c.In, err)
	}
	jsonWant, _ := jsonutils.ParseString(c.Want)
	if !reflect.DeepEqual(jsonGot, jsonWant) {
		t.Errorf("json not equal, want %s, got %s",
			jsonWant.String(), jsonGot.String())
	}
}

func testSs(t *testing.T, cs []*S) {
	for _, c := range cs {
		testS(t, c)
	}
}

func TestOptionsStructToParams(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		type s struct {
			Int int
		}
		cases := []*S{
			{
				In:   &s{100},
				Want: "{int: 100}",
			},
			{
				In:   &s{0},
				Want: "{int: 0}",
			},
		}
		testSs(t, cases)
	})
	t.Run("int ptr", func(t *testing.T) {
		type s struct {
			IntP *int
		}
		cases := []*S{
			{
				In:   &s{},
				Want: "{}",
			},
			{
				In:   &s{Int(100)},
				Want: "{int_p: 100}",
			},
			{
				In:   &s{Int(0)},
				Want: "{int_p: 0}",
			},
		}
		testSs(t, cases)
	})
	t.Run("bool", func(t *testing.T) {
		type s struct {
			Bool bool
		}
		cases := []*S{
			{
				In:   &s{},
				Want: "{bool: false}",
			},
			{
				In:   &s{true},
				Want: "{bool: true}",
			},
			{
				In:   &s{false},
				Want: "{bool: false}",
			},
		}
		testSs(t, cases)
	})
	t.Run("bool ptr", func(t *testing.T) {
		type s struct {
			BoolP *bool
		}
		cases := []*S{
			{
				In:   &s{},
				Want: "{}",
			},
			{
				In:   &s{Bool(true)},
				Want: "{bool_p: true}",
			},
			{
				In:   &s{Bool(false)},
				Want: "{bool_p: false}",
			},
		}
		testSs(t, cases)
	})
	t.Run("string", func(t *testing.T) {
		type s struct {
			String string
		}
		cases := []*S{
			{
				In:   &s{},
				Want: `{}`,
			},
			{
				In:   &s{""},
				Want: `{}`,
			},
			{
				In:   &s{"holy"},
				Want: `{string: "holy"}`,
			},
		}
		testSs(t, cases)
	})
	t.Run("string ptr", func(t *testing.T) {
		type s struct {
			String *string `json:",allowempty"`
		}
		cases := []*S{
			{
				In:   &s{},
				Want: `{}`,
			},
			{
				In:   &s{String("")},
				Want: `{"string": ""}`,
			},
			{
				In:   &s{String("holy")},
				Want: `{string: "holy"}`,
			},
		}
		testSs(t, cases)
	})
	t.Run("string slice", func(t *testing.T) {
		type s struct {
			StringSlice []string
		}
		cases := []*S{
			{
				In:   &s{},
				Want: `{}`,
			},
			{
				In:   &s{[]string{}},
				Want: `{}`,
			},
			{
				In:   &s{[]string{"holy"}},
				Want: `{"string_slice": "holy"}`,
			},
			{
				In:   &s{[]string{"holy", "goblet"}},
				Want: `{"string_slice.0": "holy", "string_slice.1": "goblet"}`,
			},
		}
		testSs(t, cases)
	})
	t.Run("json tag", func(t *testing.T) {
		type s struct {
			StringSlice []string `json:"string"`
		}
		cases := []*S{
			{
				In:   &s{},
				Want: `{}`,
			},
			{
				In:   &s{[]string{}},
				Want: `{}`,
			},
			{
				In:   &s{[]string{"holy"}},
				Want: `{"string": "holy"}`,
			},
			{
				In:   &s{[]string{"holy", "goblet"}},
				Want: `{"string.0": "holy", "string.1": "goblet"}`,
			},
		}
		testSs(t, cases)
	})
	t.Run("json tag ignore", func(t *testing.T) {
		type s struct {
			StringSliceIgnored []string `json:"-"`
		}
		cases := []*S{
			{
				In:   &s{},
				Want: `{}`,
			},
			{
				In:   &s{[]string{}},
				Want: `{}`,
			},
			{
				In:   &s{[]string{"holy"}},
				Want: `{}`,
			},
			{
				In:   &s{[]string{"holy", "goblet"}},
				Want: `{}`,
			},
		}
		testSs(t, cases)
	})
}

func TestBaseListOptions(t *testing.T) {
	t.Run("pending-delete-all", func(t *testing.T) {
		opts := &BaseListOptions{
			PendingDeleteAll: Bool(true),
		}
		params, err := opts.Params()
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		for _, f := range []string{"details"} {
			got, err := params.Bool(f)
			if got {
				t.Fatalf("pending_delete=all should not imply details=true: %v", err)
			}
		}
	})
}
