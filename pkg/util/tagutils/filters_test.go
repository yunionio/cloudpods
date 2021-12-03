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

package tagutils

import (
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"
)

func TestTagSetToFilters(t *testing.T) {
	cases := []struct {
		ts  TTagSet
		pos map[string][]string
		neg map[string][]string
	}{
		{
			ts: TTagSet{
				STag{
					Key:   "project",
					Value: "a",
				},
			},
			pos: map[string][]string{
				"project": {
					"a",
				},
			},
			neg: nil,
		},
		{
			ts: TTagSet{
				STag{
					Key:   "project",
					Value: NoValue,
				},
			},
			pos: map[string][]string{},
			neg: map[string][]string{
				"project": {},
			},
		},
		{
			ts: TTagSet{
				STag{
					Key:   "env",
					Value: "c",
				},
				STag{
					Key:   "project",
					Value: NoValue,
				},
			},
			pos: map[string][]string{
				"env": {
					"c",
				},
			},
			neg: map[string][]string{
				"env": {
					"c",
				},
				"project": {},
			},
		},
	}
	for i, c := range cases {
		gotPos, gotNeg := c.ts.toFilters()
		if !reflect.DeepEqual(c.pos, gotPos) {
			t.Errorf("[%d] filters want: %s got: %s", i, jsonutils.Marshal(c.pos), jsonutils.Marshal(gotPos))
		}
		if !reflect.DeepEqual(c.neg, gotNeg) {
			t.Errorf("[%d] nofilters want %s got %s", i, jsonutils.Marshal(c.neg), jsonutils.Marshal(gotNeg))
		}
	}
}
