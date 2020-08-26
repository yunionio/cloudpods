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
	"testing"
)

func TestHasSufffixIgnoreCase(t *testing.T) {
	cases := []struct {
		Hystack string
		Needle  string
		Want    bool
	}{
		{
			Hystack: "Hyperscacle:ASC",
			Needle:  ":ASC",
			Want:    true,
		},
		{
			Hystack: "Hyperscacle:ASC",
			Needle:  ":asc",
			Want:    true,
		},
		{
			Hystack: "Hyperscacle:ASC",
			Needle:  "ASCD",
			Want:    false,
		},
	}

	for _, c := range cases {
		got := HasSuffixIgnoreCase(c.Hystack, c.Needle)
		if got != c.Want {
			t.Errorf("HasSuffixIgnoreCase %s %s want %v got %v", c.Hystack, c.Needle, c.Want, got)
		}
	}
}

func TestHasPrefixIgnoreCase(t *testing.T) {
	cases := []struct {
		Hystack string
		Needle  string
		Want    bool
	}{
		{
			Hystack: "Hyperscacle:ASC",
			Needle:  "Hyper",
			Want:    true,
		},
		{
			Hystack: "Hyperscacle:ASC",
			Needle:  "HYPER",
			Want:    true,
		},
		{
			Hystack: "Hyperscacle:ASC",
			Needle:  "ASCD",
			Want:    false,
		},
	}

	for _, c := range cases {
		got := HasPrefixIgnoreCase(c.Hystack, c.Needle)
		if got != c.Want {
			t.Errorf("HasPrefixIgnoreCase %s %s want %v got %v", c.Hystack, c.Needle, c.Want, got)
		}
	}
}
