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

func (a TTagSet) Contains(b TTagSet) bool {
	aNoB, _, _ := Split(a, b)
	if len(aNoB) == 0 {
		return true
	} else {
		return false
	}
}

func (a TTagSet) Equals(b TTagSet) bool {
	aNoB, _, bNoA := Split(a, b)
	if len(aNoB) == 0 && len(bNoA) == 0 {
		return true
	} else {
		return false
	}
}

func Split(a, b TTagSet) (aNoB TTagSet, aAndB TTagSet, bNoA TTagSet) {
	a_b := make([]STag, 0)
	b_a := make([]STag, 0)
	anb := make([]STag, 0)
	i := 0
	j := 0
	for i < len(a) && j < len(b) {
		switch Compare(a[i], b[j]) {
		case 0:
			anb = append(anb, a[i])
			i += 1
			j += 1
		case -1:
			a_b = append(a_b, a[i])
			i += 1
		case 1:
			b_a = append(b_a, b[j])
			j += 1
		}
	}
	if i < len(a) {
		a_b = append(a_b, a[i:]...)
	}
	if j < len(b) {
		b_a = append(b_a, b[j:]...)
	}
	aNoB = a_b
	aAndB = anb
	bNoA = b_a
	return
}

func (t1 TTagSet) Add(t2 TTagSet) TTagSet {
	ret := TTagSet{}
	for k, v := range t1 {
		ret[k] = v
	}
	for k, v := range t2 {
		ret[k] = v
	}
	return ret
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
			tags[tag.Key] = append(tags[tag.Key], tag.Value)
		}
	}
	return tags
}
