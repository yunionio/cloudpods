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

package reflectutils

import (
	"fmt"
)

const (
	TAG_AMBIGUOUS_PREFIX  = "yunion-ambiguous-prefix"
	TAG_DEPRECATED_BY     = "yunion-deprecated-by"
	TAG_OLD_DEPRECATED_BY = "deprecated-by"
)

func expandAmbiguousPrefix(fields SStructFieldValueSet) SStructFieldValueSet {
	keyIndexMap := make(map[string][]int)
	for i := range fields {
		if fields[i].Info.Ignore {
			continue
		}
		key := fields[i].Info.MarshalName()
		values, ok := keyIndexMap[key]
		if !ok {
			values = make([]int, 0, 2)
		}
		keyIndexMap[key] = append(values, i)
	}
	for _, indexes := range keyIndexMap {
		if len(indexes) > 1 {
			// ambiguous found
			for _, idx := range indexes {
				if amPrefix, ok := fields[idx].Info.Tags[TAG_AMBIGUOUS_PREFIX]; ok {
					fields[idx].Info.Name = fmt.Sprintf("%s%s", amPrefix, fields[idx].Info.Name)
					if depBy, ok := fields[idx].Info.Tags[TAG_DEPRECATED_BY]; ok {
						fields[idx].Info.Tags[TAG_DEPRECATED_BY] = fmt.Sprintf("%s%s", amPrefix, depBy)
					}
					if depBy, ok := fields[idx].Info.Tags[TAG_OLD_DEPRECATED_BY]; ok {
						fields[idx].Info.Tags[TAG_OLD_DEPRECATED_BY] = fmt.Sprintf("%s%s", amPrefix, depBy)
					}
				}
			}
		}
	}
	return fields
}
