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

package sortedstring

import (
	"sort"
)

type SSortedStrings []string

func NewSortedStrings(strs []string) SSortedStrings {
	if strs == nil {
		return SSortedStrings{}
	}
	sort.Strings(strs)
	return SSortedStrings(strs)
}

func Append(ss SSortedStrings, ele ...string) SSortedStrings {
	ss = ss.Append(ele...)
	return ss
}

func (ss SSortedStrings) Append(ele ...string) SSortedStrings {
	if ss == nil {
		ss = NewSortedStrings([]string{})
	}
	for _, e := range ele {
		pos, find := ss.Index(e)
		if find {
			continue
		}
		ss = append(ss, e)
		copy(ss[pos+1:], ss[pos:])
		ss[pos] = e
	}
	return ss
}

func (ss SSortedStrings) Remove(ele ...string) SSortedStrings {
	if ss == nil {
		return ss
	}
	for _, e := range ele {
		pos, find := ss.Index(e)
		if !find {
			continue
		}
		if pos < len(ss)-1 {
			copy(ss[pos:], ss[pos+1:])
		}
		ss = ss[:len(ss)-1]
	}
	return ss
}

func (ss SSortedStrings) Index(needle string) (int, bool) {
	i := 0
	j := len(ss) - 1
	for i <= j {
		m := (i + j) / 2
		if ss[m] < needle {
			i = m + 1
		} else if ss[m] > needle {
			j = m - 1
		} else {
			return m, true
		}
	}
	return j + 1, false
}

func (ss SSortedStrings) Contains(needle string) bool {
	_, find := ss.Index(needle)
	return find
}

func (ss SSortedStrings) ContainsAny(needles ...string) bool {
	for i := range needles {
		_, find := ss.Index(needles[i])
		if find {
			return true
		}
	}
	return false
}

func (ss SSortedStrings) ContainsAll(needles ...string) bool {
	for i := range needles {
		_, find := ss.Index(needles[i])
		if !find {
			return false
		}
	}
	return true
}

func Contains(a, b SSortedStrings) bool {
	_, _, bNoA := Split(a, b)
	if len(bNoA) == 0 {
		return true
	} else {
		return false
	}
}

func Equals(a, b SSortedStrings) bool {
	aNoB, _, bNoA := Split(a, b)
	if len(aNoB) == 0 && len(bNoA) == 0 {
		return true
	} else {
		return false
	}
}

func Split(a, b SSortedStrings) (aNoB SSortedStrings, aAndB SSortedStrings, bNoA SSortedStrings) {
	a_b := make([]string, 0)
	b_a := make([]string, 0)
	anb := make([]string, 0)
	i := 0
	j := 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			anb = append(anb, a[i])
			i += 1
			j += 1
		} else if a[i] < b[j] {
			a_b = append(a_b, a[i])
			i += 1
		} else if a[i] > b[j] {
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
	aNoB = SSortedStrings(a_b)
	aAndB = SSortedStrings(anb)
	bNoA = SSortedStrings(b_a)
	return
}

func Merge(a, b SSortedStrings) SSortedStrings {
	ret := make([]string, 0)
	i := 0
	j := 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			ret = append(ret, a[i])
			i += 1
			j += 1
		} else if a[i] < b[j] {
			ret = append(ret, a[i])
			i += 1
		} else if a[i] > b[j] {
			ret = append(ret, b[j])
			j += 1
		}
	}
	if i < len(a) {
		ret = append(ret, a[i:]...)
	}
	if j < len(b) {
		ret = append(ret, b[j:]...)
	}
	return SSortedStrings(ret)
}

func Intersect(a, b SSortedStrings) SSortedStrings {
	ret := make([]string, 0)
	i := 0
	j := 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			ret = append(ret, a[i])
			i += 1
			j += 1
		} else if a[i] < b[j] {
			i += 1
		} else if a[i] > b[j] {
			j += 1
		}
	}
	return SSortedStrings(ret)
}
