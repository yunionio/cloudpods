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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type TTagSet []STag

func (t TTagSet) String() string {
	return jsonutils.Marshal(t).String()
}

func (t TTagSet) IsZero() bool {
	return len(t) == 0
}

func (ts TTagSet) Index(needle STag) (int, bool) {
	i := 0
	j := len(ts) - 1
	for i <= j {
		m := (i + j) / 2
		cmp := Compare(ts[m], needle)
		if cmp < 0 {
			i = m + 1
		} else if cmp > 0 {
			j = m - 1
		} else {
			return m, true
		}
	}
	return j + 1, false
}

func (ts TTagSet) Append(ele ...STag) TTagSet {
	for _, e := range ele {
		pos, find := ts.Index(e)
		if find {
			continue
		}
		ts = append(ts, e)
		copy(ts[pos+1:], ts[pos:])
		ts[pos] = e
	}
	return ts
}

func (ts TTagSet) Remove(ele ...STag) TTagSet {
	if len(ts) == 0 {
		return ts
	}
	for _, e := range ele {
		pos, find := ts.Index(e)
		if !find {
			continue
		}
		if pos < len(ts)-1 {
			copy(ts[pos:], ts[pos+1:])
		}
		ts = ts[:len(ts)-1]
	}
	return ts
}

func contains(v1, v2 []string) bool {
	vv1 := stringutils2.SSortedStrings(v1)
	vv2 := stringutils2.SSortedStrings(v2)
	return stringutils2.Contains(vv1, vv2)
}

func (a TTagSet) Contains(b TTagSet) bool {
	mapA := Tagset2Map(a)
	mapB := Tagset2Map(b)

	for k, v := range mapA {
		if vs, ok := mapB[k]; !ok || !contains(v, vs) {
			return false
		}
	}
	return true
}

func Map2Tagset(meta map[string]string) TTagSet {
	ts := TTagSet{}
	for k, v := range meta {
		ts = ts.Append(STag{
			Key:   k,
			Value: v,
		})
	}
	return ts
}

func Tagset2Map(oTags TTagSet) map[string][]string {
	tags := map[string][]string{}
	for _, tag := range oTags {
		if _, ok := tags[tag.Key]; !ok {
			tags[tag.Key] = []string{}
		}
		if len(tag.Value) > 0 && !utils.IsInStringArray(tag.Value, tags[tag.Key]) {
			tags[tag.Key] = stringutils2.SSortedStrings(tags[tag.Key]).Append(tag.Value)
		}
	}
	return tags
}

func Tagset2MapString(oTags TTagSet) map[string]string {
	tags := map[string]string{}
	for _, tag := range oTags {
		if _, ok := tags[tag.Key]; !ok {
			tags[tag.Key] = tag.Value
		}
	}
	return tags
}
