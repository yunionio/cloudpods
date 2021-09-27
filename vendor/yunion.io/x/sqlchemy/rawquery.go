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

// SRawQueryField is a struct represents a field of a raw SQL query
// a raw query is a query that not follow standard SELECT ... FROM ... pattern
// e.g. show tables
// the struct implements IQueryField interface
type SRawQueryField struct {
	name string
}

// Expression implementation of SRawQueryField for IQueryField
func (rqf *SRawQueryField) Expression() string {
	return rqf.name
}

// Name implementation of SRawQueryField for IQueryField
func (rqf *SRawQueryField) Name() string {
	return rqf.name
}

// Reference implementation of SRawQueryField for IQueryField
func (rqf *SRawQueryField) Reference() string {
	return rqf.name
}

// Label implementation of SRawQueryField for IQueryField
func (rqf *SRawQueryField) Label(label string) IQueryField {
	return rqf
}

// Variables implementation of SRawQueryField for IQueryField
func (rqf *SRawQueryField) Variables() []interface{} {
	return nil
}

// NewRawQuery returns an instance of SQuery with raw SQL query. e.g. show tables
func NewRawQuery(sqlStr string, fields ...string) *SQuery {
	qfs := make([]IQueryField, len(fields))
	for i, f := range fields {
		rqf := SRawQueryField{name: f}
		qfs[i] = &rqf
	}
	q := SQuery{rawSql: sqlStr, fields: qfs}
	return &q
}
