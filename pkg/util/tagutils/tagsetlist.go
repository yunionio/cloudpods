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
	"sort"
	"strings"
)

type TTagSetList []TTagSet

func (t1 TTagSetList) Contains(t2 TTagSet) bool {
	for _, ts1 := range t1 {
		if ts1.Contains(t2) {
			return true
		}
	}
	return false
}

// Contains of TTagSetList
//    tagsetlist t1 contains tagsetlist t2 means any tag set of t2 is
//    contained by one of the tag set of t1
func (t1 TTagSetList) ContainsAll(t2 TTagSetList) bool {
	if len(t2) == 0 {
		return true
	}
	for _, ts2 := range t2 {
		if len(t1) == 0 {
			return false
		}
		contained := false
		for _, ts1 := range t1 {
			if ts1.Contains(ts2) {
				contained = true
				break
			}
		}
		if !contained {
			return false
		}
	}
	return true
}

func (tsl TTagSetList) Append(t TTagSet) TTagSetList {
	ret := TTagSetList{}
	for i := 0; i < len(tsl); i++ {
		if t != nil && t.Contains(tsl[i]) {
			// skip append
		} else {
			if t != nil && tsl[i].Contains(t) {
				t = nil
			}
			ret = append(ret, tsl[i])
		}
	}
	if t != nil {
		ret = append(ret, t)
	}
	return ret
}

func (tsl TTagSetList) String() string {
	tss := make([]string, len(tsl))
	for i := 0; i < len(tss); i++ {
		tss[i] = tsl[i].String()
	}
	sort.Strings(tss)
	return "[" + strings.Join(tss, ",") + "]"
}

func (tsl TTagSetList) Flattern() TTagSet {
	ret := TTagSet{}
	for _, ts := range tsl {
		for k, v := range ts {
			ret[k] = v
		}
	}
	return ret
}
