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

package clickhouse

import (
	"fmt"

	"yunion.io/x/sqlchemy"
)

// GROUP_CONCAT1 represents the SQL function GROUP_CONCAT
func (click *SClickhouseBackend) GROUP_CONCAT2(name string, sep string, field sqlchemy.IQueryField) sqlchemy.IQueryField {
	return sqlchemy.NewFunctionField(name, true, fmt.Sprintf("arrayStringConcat(groupUniqArray(%%s), '%s')", sep), field)
}

// cast field to string
func (click *SClickhouseBackend) CASTString(field sqlchemy.IQueryField, fieldname string) sqlchemy.IQueryField {
	return sqlchemy.NewFunctionField(fieldname, false, `CAST(%s, 'String')`, field)
}

// cast field to integer
func (click *SClickhouseBackend) CASTInt(field sqlchemy.IQueryField, fieldname string) sqlchemy.IQueryField {
	return sqlchemy.NewFunctionField(fieldname, false, `CAST(%s, 'Int64')`, field)
}

// cast field to float
func (click *SClickhouseBackend) CASTFloat(field sqlchemy.IQueryField, fieldname string) sqlchemy.IQueryField {
	return sqlchemy.NewFunctionField(fieldname, false, `CAST(%s, 'Float64')`, field)
}
