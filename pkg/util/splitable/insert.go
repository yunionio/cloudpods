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
	"fmt"
	"reflect"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/sqlchemy"
)

func (t *SSplitTableSpec) Insert(dt interface{}) error {
	metas, err := t.GetTableMetas()
	if err != nil {
		return errors.Wrap(err, "GetTableMeta")
	}
	var lastDate time.Time
	vs := reflectutils.FetchAllStructFieldValueSet(reflect.Indirect(reflect.ValueOf(dt)))
	if lastDateV, ok := vs.GetValue(t.dateField); !ok {
		return errors.Wrap(errors.ErrInvalidStatus, "no dateField found")
	} else {
		lastDate = lastDateV.Interface().(time.Time)
	}

	var lastRecIndex int64
	var lastRecDate time.Time
	var lastTableSpec *sqlchemy.STableSpec

	newMeta := false
	if len(metas) > 0 {
		lastMeta := metas[len(metas)-1]
		if !lastMeta.StartDate.IsZero() && lastDate.Sub(lastMeta.StartDate) > t.maxDuration {
			lastTable := t.GetTableSpec(lastMeta)
			ti := lastTable.Instance()
			q := ti.Query(sqlchemy.MAX("last_index", ti.Field(t.indexField)), sqlchemy.MAX("last_date", ti.Field(t.dateField)))
			r := q.Row()
			err := r.Scan(&lastRecIndex, &lastRecDate)
			if err != nil {
				return errors.Wrap(err, "scan lastRecIndex and lastRecDate")
			}
			// seal last meta
			_, err = t.metaSpec.Update(&lastMeta, func() error {
				lastMeta.End = lastRecIndex
				lastMeta.EndDate = lastRecDate
				return nil
			})
			if err != nil {
				return errors.Wrap(err, "Update last meta")
			}
			newMeta = true
		} else {
			if lastMeta.StartDate.IsZero() {
				indexCol := t.tableSpec.ColumnSpec(t.indexField)
				_, err = t.metaSpec.Update(&lastMeta, func() error {
					lastMeta.Start = indexCol.(*sqlchemy.SIntegerColumn).AutoIncrementOffset
					lastMeta.StartDate = lastDate
					return nil
				})
				if err != nil {
					return errors.Wrap(err, "Update last meta")
				}
			}
			lastTableSpec = t.GetTableSpec(lastMeta)
		}
	} else {
		newMeta = true
	}
	if newMeta {
		lastTableSpec, err = t.newTable(lastRecIndex, lastDate)
		if err != nil {
			return errors.Wrap(err, "newTable")
		}
	}
	return lastTableSpec.Insert(dt)
}

func (t *SSplitTableSpec) newTable(lastRecIndex int64, lastDate time.Time) (*sqlchemy.STableSpec, error) {
	// insert a new metadata
	meta := STableMetadata{
		Table: fmt.Sprintf("%s_%d", t.tableName, lastDate.Unix()),
	}
	if lastRecIndex > 0 {
		meta.Start = lastRecIndex + 1
		meta.StartDate = lastDate
	}
	err := t.metaSpec.Insert(&meta)
	if err != nil {
		return nil, errors.Wrap(err, "insert new meta")
	}
	// create new table
	newTable := t.GetTableSpec(meta)
	err = newTable.Sync()
	if err != nil {
		return nil, errors.Wrap(err, "sync new table")
	}
	return newTable, nil
}
