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

type SRawQueryField struct {
	name string
}

func (rqf *SRawQueryField) Expression() string {
	return rqf.name
}

func (rqf *SRawQueryField) Name() string {
	return rqf.name
}

func (rqf *SRawQueryField) Reference() string {
	return rqf.name
}

func (rqf *SRawQueryField) Label(label string) IQueryField {
	return rqf
}

func NewRawQuery(sqlStr string, fields ...string) *SQuery {
	qfs := make([]IQueryField, len(fields))
	for i, f := range fields {
		rqf := SRawQueryField{name: f}
		qfs[i] = &rqf
	}
	q := SQuery{rawSql: sqlStr, fields: qfs}
	return &q
}
