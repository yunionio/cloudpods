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

type STagFilters struct {
	Filters   []map[string][]string
	NoFilters []map[string][]string
}

func splitValues(values []string) (bool, []string) {
	ret := make([]string, 0, len(values))
	if len(values) > 0 && values[len(values)-1] == NoValue {
		ret = append(ret, values[:len(values)-1]...)
		return true, ret
	} else {
		ret = append(ret, values...)
		return false, ret
	}
}

func (tf *STagFilters) AddFilter(ts TTagSet) {
	tf.Filters = append(tf.Filters, tagset2Map(ts))
}

func (tf *STagFilters) AddNoFilter(ts TTagSet) {
	tf.NoFilters = append(tf.NoFilters, tagset2Map(ts))
}

func (tf *STagFilters) AddFilters(tsl TTagSetList) {
	for _, ts := range tsl {
		tf.AddFilter(ts)
	}
}

func (tf *STagFilters) AddNoFilters(tsl TTagSetList) {
	for _, ts := range tsl {
		tf.AddNoFilter(ts)
	}
}
