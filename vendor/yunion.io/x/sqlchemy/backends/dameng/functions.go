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

package dameng

import (
	"fmt"

	"yunion.io/x/sqlchemy"
)

// GROUP_CONCAT2 represents the SQL function GROUP_CONCAT
func (dameng *SDamengBackend) GROUP_CONCAT2(name string, sep string, field sqlchemy.IQueryField) sqlchemy.IQueryField {
	if sep == "," {
		return sqlchemy.NewFunctionField(name, true, `WM_CONCAT(%s)`, field)
	} else {
		return sqlchemy.NewFunctionField(name, true, fmt.Sprintf("REPLACE(WM_CONCAT(%%s), ',', '%s')", sep), field)
	}
}

// INET_ATON represents the SQL function INET_ATON
func (dameng *SDamengBackend) INET_ATON(field sqlchemy.IQueryField) sqlchemy.IQueryField {
	expr := ""
	vars := make([]sqlchemy.IQueryField, 0)
	expr += `TO_NUMBER(SUBSTR(%s,1,INSTR(%s,'.')-1))*POWER(256,3)+`
	vars = append(vars, field, field)
	expr += `TO_NUMBER(SUBSTR(%s,INSTR(%s,'.')+1,INSTR(%s,'.',1,2)-INSTR(%s,'.')-1))*POWER(256,2)+`
	vars = append(vars, field, field, field, field)
	expr += `TO_NUMBER(SUBSTR(%s,INSTR(%s,'.',1,2)+1,INSTR(%s,'.',1,3)-INSTR(%s,'.',1,2)-1))*256+`
	vars = append(vars, field, field, field, field)
	expr += `TO_NUMBER(SUBSTR(%s,INSTR(%s,'.',1,3)+1))`
	vars = append(vars, field, field)
	return sqlchemy.NewFunctionField("", false, expr, vars...)
}

// cast field to string
func (dameng *SDamengBackend) CASTString(field sqlchemy.IQueryField, fieldname string) sqlchemy.IQueryField {
	return sqlchemy.NewFunctionField(fieldname, false, `CAST(%s AS VARCHAR)`, field)
}

// cast field to integer
func (dameng *SDamengBackend) CASTInt(field sqlchemy.IQueryField, fieldname string) sqlchemy.IQueryField {
	return sqlchemy.NewFunctionField(fieldname, false, `CAST(%s AS INTEGER)`, field)
}

// cast field to float
func (dameng *SDamengBackend) CASTFloat(field sqlchemy.IQueryField, fieldname string) sqlchemy.IQueryField {
	return sqlchemy.NewFunctionField(fieldname, false, `CAST(%s AS REAL)`, field)
}
