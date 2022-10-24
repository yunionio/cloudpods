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

package splitable

import (
	"database/sql"
	"fmt"
	"reflect"
	"sync"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"
)

type SSplitTableSpec struct {
	indexField  string
	dateField   string
	tableName   string
	tableSpec   *sqlchemy.STableSpec
	metaSpec    *sqlchemy.STableSpec
	maxDuration time.Duration
	maxSegments int

	lastTableSpec   *sqlchemy.STableSpec
	lastTableLock   *sync.Mutex
	lastTableExpire time.Time
}

const (
	lastTableSpecExpireHours = 1 // 1 hour
)

func (t *SSplitTableSpec) DataType() reflect.Type {
	return t.tableSpec.DataType()
}

func (t *SSplitTableSpec) ColumnSpec(name string) sqlchemy.IColumnSpec {
	return t.tableSpec.ColumnSpec(name)
}

func (t *SSplitTableSpec) Name() string {
	return t.tableName
}

func (t *SSplitTableSpec) Database() *sqlchemy.SDatabase {
	return t.metaSpec.Database()
}

func (t *SSplitTableSpec) Columns() []sqlchemy.IColumnSpec {
	return t.tableSpec.Columns()
}

func (t *SSplitTableSpec) PrimaryColumns() []sqlchemy.IColumnSpec {
	return t.tableSpec.PrimaryColumns()
}

func (t *SSplitTableSpec) Indexes() []sqlchemy.STableIndex {
	return t.tableSpec.Indexes()
}

func (t *SSplitTableSpec) Expression() string {
	metas, err := t.GetTableMetas()
	if err != nil {
		return fmt.Sprintf("`%s`", t.tableName)
	}
	tss := make([]sqlchemy.IQuery, 0)
	for _, meta := range metas {
		ts := t.GetTableSpec(meta)
		tss = append(tss, ts.Query())
	}
	union, err := sqlchemy.UnionWithError(tss...)
	if err != nil {
		return fmt.Sprintf("`%s`", t.tableName)
	}
	return union.Expression()
}

func (t *SSplitTableSpec) Instance() *sqlchemy.STable {
	return sqlchemy.NewTableInstance(t)
}

func (t *SSplitTableSpec) DropForeignKeySQL() []string {
	return t.tableSpec.DropForeignKeySQL()
}

func (t *SSplitTableSpec) AddIndex(unique bool, cols ...string) bool {
	return t.tableSpec.AddIndex(unique, cols...)
}

func (t *SSplitTableSpec) Fetch(dt interface{}) error {
	vs := reflectutils.FetchStructFieldValueSet(reflect.Indirect(reflect.ValueOf(dt)))
	idxVal, ok := vs.GetValue(t.indexField)
	if !ok {
		return errors.Wrap(errors.ErrNotFound, "GetValue")
	}
	idxInt := idxVal.Int()
	metas, err := t.GetTableMetas()
	if err != nil {
		return errors.Wrap(err, "GetTableMetas")
	}
	for _, meta := range metas {
		if idxInt >= meta.Start && (meta.End == 0 || meta.End >= idxInt) {
			ts := t.GetTableSpec(meta)
			return ts.Fetch(dt)
		}
	}
	return sql.ErrNoRows
}

func (t *SSplitTableSpec) Drop() error {
	metas, err := t.GetTableMetas()
	if err != nil {
		return errors.Wrap(err, "GetTableMetas")
	}
	for _, meta := range metas {
		ts := t.GetTableSpec(meta)
		err := ts.Drop()
		if err != nil {
			return errors.Wrapf(err, "Drop %s", meta.Table)
		}
	}
	err = t.metaSpec.Drop()
	if err != nil {
		return errors.Wrap(err, "Drop Meta")
	}
	return nil
}

func NewSplitTableSpec(s interface{}, name string, indexField string, dateField string, maxDuration time.Duration, maxSegments int, dbName sqlchemy.DBName) (*SSplitTableSpec, error) {
	spec := sqlchemy.NewTableSpecFromStructWithDBName(s, name, dbName)
	/*indexCol := spec.ColumnSpec(indexField)
	if indexCol == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "indexField %s not found", indexField)
	}
	if !indexCol.IsPrimary() {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "indexField %s not primary", indexField)
	}
	if !indexCol.IsAutoIncrement() {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "indexField %s not auto_increment", indexField)
	}
	dateCol := spec.ColumnSpec(dateField)
	if dateCol == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "dateField %s not found", dateField)
	}
	if !dateCol.IsDateTime() {
		return nil, errors.Wrapf(errors.ErrInvalidStatus, "dateField %s not datetime column", dateField)
	}*/

	metaSpec := sqlchemy.NewTableSpecFromStructWithDBName(&STableMetadata{}, fmt.Sprintf("%s_metadata", name), dbName)

	sts := &SSplitTableSpec{
		indexField:  indexField,
		dateField:   dateField,
		tableName:   name,
		tableSpec:   spec,
		metaSpec:    metaSpec,
		maxDuration: maxDuration,
		maxSegments: maxSegments,

		lastTableLock: &sync.Mutex{},
	}

	registerSplitable(sts)

	return sts, nil
}
