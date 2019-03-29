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

package excelutils

import "testing"

func arrayEqual(a1, a2 []int) bool {
	if len(a1) != len(a2) {
		return false
	}
	for i := 0; i < len(a1); i += 1 {
		if a1[i] != a2[i] {
			return false
		}
	}
	return true
}

func TestDecimalBaseMaxWidth(t *testing.T) {
	cases := []struct {
		decIn  int
		baseIn int
		want   int
	}{
		{100, 10, 3},
		{16, 16, 2},
		{15, 16, 1},
	}
	for _, c := range cases {
		got := decimalBaseMaxWidth(c.decIn, c.baseIn)
		if got != c.want {
			t.Errorf("decimalBaseMaxWidth(%d, %d) = %d != %d", c.decIn, c.baseIn, got, c.want)
		}
	}

	cases2 := []struct {
		decIn  int
		baseIn int
		width  int
		want   int
		want2  int
	}{
		{100, 10, 3, 1, 100},
		{16, 16, 2, 1, 16},
		{15, 16, 1, 15, 1},
	}

	for _, c := range cases2 {
		got1, got2 := decimalBaseN(c.decIn, c.baseIn, c.width)
		if got1 != c.want || got2 != c.want2 {
			t.Errorf("decimalBaseN(%d %d %d) = %d %d != %d %d", c.decIn, c.baseIn, c.width, got1, got2, c.want, c.want2)
		}
	}

	cases3 := []struct {
		decIn  int
		baseIn int
		want   []int
	}{
		{100, 10, []int{1, 0, 0}},
		{16, 16, []int{1, 0}},
		{0, 16, []int{0}},
		{15, 16, []int{15}},
		{0, 26, []int{0}},
		{1, 26, []int{1}},
		{25, 26, []int{25}},
		{26, 26, []int{1, 0}},
		{27, 26, []int{1, 1}},
		{676, 26, []int{1, 0, 0}},
	}
	for _, c := range cases3 {
		got := decimal2Base(c.decIn, c.baseIn)
		if !arrayEqual(got, c.want) {
			t.Errorf("decimal2Base(%d %d) = %#v != %#v", c.decIn, c.baseIn, got, c.want)
		}
	}

	cases4 := []struct {
		decIn int
		want  string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
		{676, "AAA"},
	}
	for _, c := range cases4 {
		got := decimal2Alphabet(c.decIn)
		if got != c.want {
			t.Errorf("decimal2Alphabet(%d) = %s != %s", c.decIn, got, c.want)
		}
	}
}
