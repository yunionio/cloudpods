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

// expandAmbiguousPrefix prepends the prefix given by the
// TAG_AMBIGUOUS_PREFIX tag to the fields sharing a name with another field.
//
// Expanding a prefix can itself collide with a name that is already in use, so
// the expansion is repeated until the names settle.  A field whose expanded
// name would take over a name owned by a field outside its own ambiguous
// group keeps its original name, so that the expansion never introduces a new
// ambiguity.
func expandAmbiguousPrefix(fields SStructFieldValueSet) SStructFieldValueSet {
	prefixed := make(map[int]bool)
	for {
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
		changed := false
		for _, indexes := range keyIndexMap {
			if len(indexes) < 2 {
				continue
			}
			// ambiguous found
			for _, idx := range indexes {
				if prefixed[idx] {
					continue
				}
				amPrefix, ok := fields[idx].Info.Tag(TAG_AMBIGUOUS_PREFIX)
				if !ok {
					continue
				}
				expanded := fmt.Sprintf("%s%s", amPrefix, fields[idx].Info.Name)
				if takenByOther(fields, expanded, indexes) {
					continue
				}
				info := fields[idx].Info
				info.Name = expanded
				_, newDepBy := info.tags[TAG_DEPRECATED_BY]
				_, oldDepBy := info.tags[TAG_OLD_DEPRECATED_BY]
				if newDepBy || oldDepBy {
					// the tags may still be shared with other callers
					info.copyTags()
				}
				if newDepBy {
					info.tags[TAG_DEPRECATED_BY] = fmt.Sprintf("%s%s", amPrefix, info.tags[TAG_DEPRECATED_BY])
				}
				if oldDepBy {
					info.tags[TAG_OLD_DEPRECATED_BY] = fmt.Sprintf("%s%s", amPrefix, info.tags[TAG_OLD_DEPRECATED_BY])
				}
				if len(info.aliases) > 0 {
					info.copyAliases()
					for i := range info.aliases {
						info.aliases[i] = fmt.Sprintf("%s%s", amPrefix, info.aliases[i])
					}
				}
				prefixed[idx] = true
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return fields
}

// takenByOther reports whether name is already used by a field outside group
func takenByOther(fields SStructFieldValueSet, name string, group []int) bool {
	for i := range fields {
		if fields[i].Info.Ignore {
			continue
		}
		if fields[i].Info.MarshalName() != name {
			continue
		}
		if !containsIndex(group, i) {
			return true
		}
	}
	return false
}

func containsIndex(indexes []int, idx int) bool {
	for _, i := range indexes {
		if i == idx {
			return true
		}
	}
	return false
}
