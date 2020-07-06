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

package stringutils2

import (
	"reflect"
	"testing"
)

func TestSortedString(t *testing.T) {
	input := []string{"Go", "Bravo", "Gopher", "Alpha", "Grin", "Delta"}
	// Alpha Bravo Delta Go Gopher Grin
	// 0     1     2     3  4      5
	ss := NewSortedStrings(input)
	cases := []struct {
		Needle string
		Index  int
		Want   bool
	}{
		{"Go", 3, true},
		{"Bravo", 1, true},
		{"Gopher", 4, true},
		{"Alpha", 0, true},
		{"Grin", 5, true},
		{"Delta", 2, true},
		{"Go1", 4, false},
		{"G", 3, false},
		{"A", 0, false},
		{"T", 6, false},
	}
	for _, c := range cases {
		idx, find := ss.Index(c.Needle)
		if idx != c.Index || find != c.Want {
			t.Errorf("%s: want: %d %v got: %d %v", c.Needle, c.Index, c.Want, idx, find)
		}
	}
}

func TestSplitStrings(t *testing.T) {
	input := []string{"Go", "Bravo", "Gopher", "Alpha", "Grin", "Delta"}
	input2 := []string{"Go2", "Bravo", "Gopher", "Alpha1", "Grin", "Delt"}

	ss1 := NewSortedStrings(input)
	ss2 := NewSortedStrings(input2)

	a, b, c := Split(ss1, ss2)
	t.Logf("A: %s", ss1)
	t.Logf("B: %s", ss2)
	t.Logf("A-B: %s", a)
	t.Logf("AnB: %s", b)
	t.Logf("B-A: %s", c)
}

func TestMergeStrings(t *testing.T) {
	input := []string{"Go", "Bravo", "Gopher", "Alpha", "Grin", "Delta"}
	input2 := []string{"Go2", "Bravo", "Gopher", "Alpha1", "Grin", "Delt"}

	ss1 := NewSortedStrings(input)
	ss2 := NewSortedStrings(input2)

	m := Merge(ss1, ss2)
	t.Logf("A: %s", ss1)
	t.Logf("B: %s", ss2)
	t.Logf("%s", m)
}

func TestSortedStringsAppend(t *testing.T) {
	cases := []struct {
		in   []string
		ele  []string
		want SSortedStrings
	}{
		{
			in:   []string{"Alpha", "Bravo", "Go"},
			ele:  []string{"Go2"},
			want: []string{"Alpha", "Bravo", "Go", "Go2"},
		},
		{
			in:   []string{"Alpha", "Bravo", "Go2"},
			ele:  []string{"Go"},
			want: []string{"Alpha", "Bravo", "Go", "Go2"},
		},
		{
			in:   []string{"Alpha", "Bravo", "Go2"},
			ele:  []string{"Aaaa", "Go"},
			want: []string{"Aaaa", "Alpha", "Bravo", "Go", "Go2"},
		},
	}
	for _, c := range cases {
		got := NewSortedStrings(c.in).Append(c.ele...)
		if !reflect.DeepEqual(c.want, got) {
			t.Errorf("want: %s got: %s", c.want, got)
		}
	}
}

func TestSortedStringsRemove(t *testing.T) {
	cases := []struct {
		in   []string
		ele  []string
		want SSortedStrings
	}{
		{
			in:   []string{"Alpha", "Bravo", "Go"},
			ele:  []string{"Go", "Go2"},
			want: []string{"Alpha", "Bravo"},
		},
		{
			in:   []string{"Alpha", "Bravo", "Go2"},
			ele:  []string{"Go"},
			want: []string{"Alpha", "Bravo", "Go2"},
		},
		{
			in:   []string{"Alpha", "Bravo", "Go", "Go2"},
			ele:  []string{"Aaaa", "Alpha"},
			want: []string{"Bravo", "Go", "Go2"},
		},
	}
	for _, c := range cases {
		got := NewSortedStrings(c.in).Remove(c.ele...)
		if !reflect.DeepEqual(c.want, got) {
			t.Errorf("want: %s got: %s", c.want, got)
		}
	}
}
