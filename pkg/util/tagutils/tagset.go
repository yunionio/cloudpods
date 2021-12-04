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

	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type TTagSet []STag

func (t TTagSet) String() string {
	return jsonutils.Marshal(t).String()
}

func (t TTagSet) IsZero() bool {
	return len(t) == 0
}

func (ts TTagSet) index(needle STag) (int, bool) {
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
		ts = ts.add(e)
	}
	return ts
}

func (ts TTagSet) add(e STag) TTagSet {
	if len(e.Value) == 0 {
		e.Value = AnyValue
	}
	pos, find := ts.index(e)
	if find {
		return ts
	}
	ts = append(ts, e)
	copy(ts[pos+1:], ts[pos:])
	ts[pos] = e
	if e.Value == AnyValue {
		for i := pos + 1; i < len(ts); i++ {
			if ts[i].Key != e.Key {
				if i > pos+1 {
					copy(ts[pos+1:], ts[i:])
					ts = ts[:len(ts)-i+pos+1]
				}
				break
			} else if ts[i].Value == NoValue {
				// remove this key completely
				copy(ts[pos:], ts[i+1:])
				ts = ts[:len(ts)-i+pos-1]
				break
			}
		}
	} else if e.Value == NoValue && pos > 0 && ts[pos-1].Key == e.Key && ts[pos-1].Value == AnyValue {
		copy(ts[pos-1:], ts[pos:])
		ts = ts[:len(ts)-1]
	}
	return ts
}

func (ts TTagSet) Remove(ele ...STag) TTagSet {
	if len(ts) == 0 {
		return ts
	}
	for _, e := range ele {
		if len(e.Value) == 0 {
			e.Value = AnyValue
		}
		pos, find := ts.index(e)
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

func (a TTagSet) Compact() TTagSet {
	ret := make(TTagSet, 0, len(a))
	ret = ret.Append(a...)
	return ret
}

func (a TTagSet) Contains(b TTagSet) bool {
	a = a.Compact()
	b = b.Compact()
	i := 0
	j := 0
	for i < len(a) && j < len(b) {
		if a[i].Key < b[j].Key {
			return false
		} else if a[i].Key > b[j].Key {
			j++
		} else {
			// a[i].Key == b[j].Key
			if a[i].Value == b[j].Value {
				i++
				j++
			} else {
				if a[i].Value == AnyValue {
					j++
					if j >= len(b) || a[i].Key != b[j].Key {
						i++
					}
				} else {
					return false
				}
			}
		}
	}
	if i < len(a) {
		return false
	}
	return true
}

func Map2Tagset(meta map[string]string) TTagSet {
	ts := TTagSet{}
	for k, v := range meta {
		ts = ts.add(STag{
			Key:   k,
			Value: v,
		})
	}
	return ts
}

func tagset2Map(oTags TTagSet) map[string][]string {
	oTags = oTags.Compact()
	tags := map[string][]string{}
	for _, tag := range oTags {
		if _, ok := tags[tag.Key]; !ok {
			tags[tag.Key] = []string{}
		}
		if tag.Value != AnyValue {
			values := stringutils2.SSortedStrings(tags[tag.Key])
			if !values.Contains(tag.Value) {
				tags[tag.Key] = append(values, tag.Value)
			}
		}
	}
	return tags
}

func Tagset2MapString(oTags TTagSet) map[string]string {
	oTags = oTags.Compact()
	tags := map[string]string{}
	for _, tag := range oTags {
		if _, ok := tags[tag.Key]; !ok {
			if tag.Value == AnyValue {
				tags[tag.Key] = ""
			} else if tag.Value == NoValue {
				// no add
			} else {
				tags[tag.Key] = tag.Value
			}
		}
	}
	return tags
}
