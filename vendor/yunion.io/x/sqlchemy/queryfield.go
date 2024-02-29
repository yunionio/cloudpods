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

import "fmt"

type sQueryField struct {
	from  IQuerySource
	name  string
	alias string
}

// the string after select
func (sqf *sQueryField) Expression() string {
	qChar := sqf.from.database().backend.QuoteChar()

	alias := sqf.name
	if len(sqf.alias) > 0 {
		alias = sqf.alias
	}
	return fmt.Sprintf("%s%s%s.%s%s%s AS %s%s%s", qChar, sqf.from.Alias(), qChar, qChar, sqf.name, qChar, qChar, alias, qChar)
}

// the name of thie field
func (sqf *sQueryField) Name() string {
	if len(sqf.alias) > 0 {
		return sqf.alias
	}
	return sqf.name
}

// the reference string in where clause
func (sqf *sQueryField) Reference() string {
	qChar := sqf.from.database().backend.QuoteChar()
	return fmt.Sprintf("%s%s%s.%s%s%s", qChar, sqf.from.Alias(), qChar, qChar, sqf.Name(), qChar)
}

// give this field an alias name
func (sqf *sQueryField) Label(label string) IQueryField {
	sqf.alias = label
	return sqf
}

// return variables
func (sqf *sQueryField) Variables() []interface{} {
	return nil
}

// Database returns the database of this IQuerySource
func (sqf *sQueryField) database() *SDatabase {
	return sqf.from.database()
}

func newQueryField(from IQuerySource, name string) *sQueryField {
	return &sQueryField{
		from: from,
		name: name,
	}
}

type queryFieldList []IQueryField

func (a queryFieldList) Len() int           { return len(a) }
func (a queryFieldList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a queryFieldList) Less(i, j int) bool { return a[i].Name() < a[j].Name() }
