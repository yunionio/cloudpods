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
	"sort"
	"strings"
)

type STableIndex struct {
	name    string
	columns []string

	isUnique bool

	ts ITableSpec
}

func NewTableIndex(ts ITableSpec, name string, cols []string, unique bool) STableIndex {
	sort.Sort(TColumnNames(cols))
	return STableIndex{
		name:    name,
		columns: cols,

		isUnique: unique,
		ts:       ts,
	}
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

const IndexLimit int = 64

func (index *STableIndex) Name() string {
	if len(index.name) > 0 {
		return index.name
	}
	name := fmt.Sprintf("ix_%s_%s", index.ts.Name(), strings.Join(index.columns, "_"))
	if len(name) > IndexLimit {
		name = name[:IndexLimit]
	}
	return name
}

func (index STableIndex) clone(ts ITableSpec) STableIndex {
	return NewTableIndex(ts, "", index.columns, index.isUnique)
}

func (index *STableIndex) IsIdentical(cols ...string) bool {
	if len(index.columns) != len(cols) {
		return false
	}
	sort.Sort(TColumnNames(cols))
	for i := 0; i < len(index.columns); i++ {
		if index.columns[i] != cols[i] {
			return false
		}
	}
	return true
}

func (index *STableIndex) QuotedColumns(quoteStr string) []string {
	ret := make([]string, len(index.columns))
	for i := 0; i < len(ret); i++ {
		ret[i] = fmt.Sprintf("%s%s%s", quoteStr, index.columns[i], quoteStr)
	}
	return ret
}

// AddIndex adds a SQL index over multiple columns for a Table
// param unique: indicates a unique index cols: name of columns
func (ts *STableSpec) addIndexWithName(name string, unique bool, cols ...string) bool {
	for i := 0; i < len(ts._indexes); i++ {
		if ts._indexes[i].IsIdentical(cols...) {
			return false
		}
	}
	idx := NewTableIndex(ts, name, cols, unique)
	ts._indexes = append(ts._indexes, idx)
	return true
}

func (ts *STableSpec) AddIndex(unique bool, cols ...string) bool {
	return ts.addIndexWithName("", unique, cols...)
}
