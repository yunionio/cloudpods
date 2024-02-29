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
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"
)

func (spec *SSplitTableSpec) Sync() error {
	err := spec.metaSpec.Sync()
	if err != nil {
		return errors.Wrap(err, "metaSpec.Sync")
	}
	metas, err := spec.GetTableMetas()
	if err != nil {
		return errors.Wrap(err, "GetTableMetas")
	}
	if len(metas) == 0 {
		// init the first metadata record
		fakeMeta := STableMetadata{
			Table: spec.tableName,
		}
		tbl := spec.GetTableSpec(fakeMeta)
		if tbl.Exists() {
			err := tbl.Sync()
			if err != nil {
				return errors.Wrap(err, "Sync")
			}
			var minIndex int64
			var minDate time.Time
			ti := tbl.Instance()
			q := ti.Query(sqlchemy.MIN("min_index", ti.Field(spec.indexField)), sqlchemy.MIN("min_date", ti.Field(spec.dateField)))
			r := q.Row()
			err = r.Scan(&minIndex, &minDate)
			if err != nil {
				return errors.Wrap(err, "minIndex minDate")
			}
			fakeMeta.Start = minIndex
			fakeMeta.StartDate = minDate
			err = spec.metaSpec.Insert(&fakeMeta)
			if err != nil {
				return errors.Wrap(err, "insert init metadata")
			}
		} else {
			_, err := spec.newTable(-1, time.Time{})
			if err != nil {
				return errors.Wrap(err, "spec.newTable")
			}
		}
	} else {
		for i := range metas {
			subSpec := spec.GetTableSpec(metas[i])
			err := subSpec.Sync()
			if err != nil {
				return errors.Wrap(err, "Sync")
			}
		}
	}
	return nil
}

func (spec *SSplitTableSpec) CheckSync() error {
	err := spec.metaSpec.CheckSync()
	if err != nil {
		return errors.Wrap(err, "metaSpec.CheckSync")
	}
	metas, err := spec.GetTableMetas()
	if err != nil {
		return errors.Wrap(err, "GetTableMetas")
	}
	if len(metas) == 0 {
		return errors.Wrap(err, "empty metadata")
	} else {
		for i := range metas {
			subSpec := spec.GetTableSpec(metas[i])
			err := subSpec.CheckSync()
			if err != nil {
				return errors.Wrap(err, "GetTableSpec")
			}
		}
	}
	return nil
}

func (spec *SSplitTableSpec) SyncSQL() []string {
	sqls := spec.metaSpec.SyncSQL()
	zeroMeta := false

	if spec.metaSpec.Exists() {
		metas, err := spec.getTableMetasForInit()
		if err != nil {
			log.Errorf("GetTableMetas fail %s", err)
			return nil
		} else if len(metas) > 0 {
			for i := range metas {
				subSpec := spec.GetTableSpec(metas[i])
				nsql := subSpec.SyncSQL()
				sqls = append(sqls, nsql...)
			}
			return sqls
		} else { // len(metas) == 0
			zeroMeta = true
		}
	} else {
		nsql := spec.metaSpec.SyncSQL()
		sqls = append(sqls, nsql...)
		zeroMeta = true
	}

	if zeroMeta {
		indexCol := spec.tableSpec.ColumnSpec(spec.indexField)
		now := time.Now()
		meta := STableMetadata{
			Table: fmt.Sprintf("%s_%d", spec.tableName, now.Unix()),
			Start: indexCol.AutoIncrementOffset(),
		}
		// insert the first meta
		insertResult, err := spec.metaSpec.InsertSqlPrep(&meta, false)
		if err != nil {
			log.Errorf("spec.metaSpec.InsertSqlPrep fail %s", err)
			return nil
		}
		// sql := fmt.Sprintf("INSERT INTO `%s`(`table`, `deleted`, `created_at`) VALUES('%s', 0, '%s')", spec.metaSpec.Name(), meta.Table, timeutils.MysqlTime(now))
		sqls = append(sqls, sqlchemy.SQLPrintf(insertResult.Sql, insertResult.Values))
		// create the first table
		newtable := spec.GetTableSpec(meta)
		nsql := newtable.SyncSQL()
		sqls = append(sqls, nsql...)
		return sqls
	}

	fakeMeta := STableMetadata{
		Table: spec.tableName,
	}
	tbl := spec.GetTableSpec(fakeMeta)
	if tbl.Exists() {
		nsql := tbl.SyncSQL()
		if len(nsql) > 0 {
			sqls = append(sqls, nsql...)
		}
		var minIndex int64
		var minDate time.Time
		ti := tbl.Instance()
		q := ti.Query(sqlchemy.MIN("min_index", ti.Field(spec.indexField)), sqlchemy.MIN("min_date", ti.Field(spec.dateField)))
		r := q.Row()
		err := r.Scan(&minIndex, &minDate)
		if err != nil {
			log.Errorf("query minIndex minDate fail %s", err)
		} else {
			minDateStr := timeutils.MysqlTime(minDate)
			sql := fmt.Sprintf("INSERT INTO `%s`(`table`, `start`, `start_date`, `deleted`, `created_at`) VALUES('%s', %d, '%s', 0, '%s')", spec.metaSpec.Name(), spec.tableName, minIndex, minDateStr, minDateStr)
			sqls = append(sqls, sql)
		}
	}
	return sqls
}
