// Copyright 2019 Yunion

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
)

// SSubQueryField represents a field of subquery, which implements IQueryField
type SSubQueryField struct {
	field IQueryField
	query *SSubQuery
	alias string
}

// Expression implementation of SSubQueryField for IQueryField
func (sqf *SSubQueryField) Expression() string {
	if len(sqf.alias) > 0 {
		return fmt.Sprintf("`%s`.`%s` AS `%s`", sqf.query.alias, sqf.field.Name(), sqf.alias)
	}
	return fmt.Sprintf("`%s`.`%s`", sqf.query.alias, sqf.field.Name())
}

// Name implementation of SSubQueryField for IQueryField
func (sqf *SSubQueryField) Name() string {
	if len(sqf.alias) > 0 {
		return sqf.alias
	}
	return sqf.field.Name()
}

// Reference implementation of SSubQueryField for IQueryField
func (sqf *SSubQueryField) Reference() string {
	return fmt.Sprintf("`%s`.`%s`", sqf.query.alias, sqf.Name())
}

// Label implementation of SSubQueryField for IQueryField
func (sqf *SSubQueryField) Label(label string) IQueryField {
	if len(label) > 0 {
		sqf.alias = label
	}
	return sqf
}

// Variables implementation of SSubQueryField for IQueryField
func (sqf *SSubQueryField) Variables() []interface{} {
	return nil
}

// database implementation of SSubQueryField for IQueryField
func (sqf *SSubQueryField) database() *SDatabase {
	return sqf.query.database()
}

// SSubQuery represents a subquery. A subquery is a query used as a query source
// SSubQuery should implementation IQuerySource
// At the same time, a subquery can be used in condition. e.g. IN condition
type SSubQuery struct {
	query IQuery
	alias string

	referedFields map[string]IQueryField
}

// Expression implementation of SSubQuery for IQuerySource
func (sq *SSubQuery) Expression() string {
	fields := make([]IQueryField, 0)
	for k := range sq.referedFields {
		fields = append(fields, sq.referedFields[k])
	}
	// Make sure the order of the fields
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name() < fields[j].Name()
	})
	return fmt.Sprintf("(%s)", sq.query.String(fields...))
}

// Alias implementation of SSubQuery for IQuerySource
func (sq *SSubQuery) Alias() string {
	return sq.alias
}

// Variables implementation of SSubQuery for IQuerySource
func (sq *SSubQuery) Variables() []interface{} {
	return sq.query.Variables()
}

func (sq *SSubQuery) findField(id string) IQueryField {
	if sq.referedFields == nil {
		sq.referedFields = make(map[string]IQueryField)
	}
	if _, ok := sq.referedFields[id]; ok {
		return sq.referedFields[id]
	}
	queryFields := sq.query.QueryFields()
	for i := range queryFields {
		if queryFields[i].Name() == id {
			sq.referedFields[id] = sq.query.Field(queryFields[i].Name())
			return sq.referedFields[id]
		}
	}
	return nil
}

// Field implementation of SSubQuery for IQuerySource
func (sq *SSubQuery) Field(id string, alias ...string) IQueryField {
	f := sq.findField(id)
	if f == nil {
		return nil
	}
	sqf := SSubQueryField{query: sq, field: f}
	if len(alias) > 0 {
		sqf.Label(alias[0])
	}
	return &sqf
}

// Fields implementation of SSubQuery for IQuerySource
func (sq *SSubQuery) Fields() []IQueryField {
	ret := make([]IQueryField, 0)
	for _, f := range sq.query.QueryFields() {
		sqf := SSubQueryField{query: sq, field: f}
		ret = append(ret, &sqf)
	}
	return ret
}

// database implementation of SSubQuery for IQuerySource
func (sq *SSubQuery) database() *SDatabase {
	return sq.query.database()
}
