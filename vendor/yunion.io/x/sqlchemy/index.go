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

package sqlchemy

import (
	"fmt"
	"strings"
)

type STableIndex struct {
	name     string
	columns  []string
	isUnique bool
}

type TColumnNames []string

func (cols TColumnNames) Len() int {
	return len(cols)
}

func (cols TColumnNames) Swap(i, j int) {
	cols[i], cols[j] = cols[j], cols[i]
}

func (cols TColumnNames) Less(i, j int) bool {
	if strings.Compare(cols[i], cols[j]) < 0 {
		return true
	} else {
		return false
	}
}

func (index *STableIndex) IsIdentical(cols ...string) bool {
	if len(index.columns) != len(cols) {
		return false
	}
	for i := 0; i < len(index.columns); i += 1 {
		if index.columns[i] != cols[i] {
			return false
		}
	}
	return true
}

func (index *STableIndex) QuotedColumns() []string {
	ret := make([]string, len(index.columns))
	for i := 0; i < len(ret); i += 1 {
		ret[i] = fmt.Sprintf("`%s`", index.columns[i])
	}
	return ret
}

func (ts *STableSpec) AddIndex(unique bool, cols ...string) bool {
	for i := 0; i < len(ts.indexes); i += 1 {
		if ts.indexes[i].IsIdentical(cols...) {
			return false
		}
	}
	name := fmt.Sprintf("ix_%s_%s", ts.name, strings.Join(cols, "_"))
	idx := STableIndex{name: name, columns: cols, isUnique: unique}
	ts.indexes = append(ts.indexes, idx)
	return true
}
