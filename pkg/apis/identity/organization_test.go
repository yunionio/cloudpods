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

package identity

import (
	"reflect"
	"testing"
)

func TestJoinLabel(t *testing.T) {
	cases := []struct {
		segs []string
		want string
	}{
		{
			segs: []string{"L1", "L2", "L3"},
			want: "L1/L2/L3",
		},
		{
			segs: []string{"L1/", "L2", "L3"},
			want: "L1/L2/L3",
		},
		{
			segs: []string{"L1/ ", "/L2", "/L3/"},
			want: "L1/L2/L3",
		},
		{
			segs: []string{"L1/ ", "/L2", "/L3/", "H4/H5"},
			want: "L1/L2/L3/H4\\/H5",
		},
	}
	for _, c := range cases {
		got := JoinLabels(c.segs...)
		if got != c.want {
			t.Errorf("got %s want %s", got, c.want)
		}
	}
}

func TestSplitLabel(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{
			in:   "L1/L2/L3",
			want: []string{"L1", "L2", "L3"},
		},
		{
			in:   "L1/L2//L3",
			want: []string{"L1", "L2", "L3"},
		},
		{
			in:   "/L1/L2/L3/",
			want: []string{"L1", "L2", "L3"},
		},
		{
			in:   "L1/L2/L3/H4\\/H5",
			want: []string{"L1", "L2", "L3", "H4/H5"},
		},
	}
	for _, c := range cases {
		got := SplitLabel(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("want %s got %s", c.want, got)
		}
	}
}
