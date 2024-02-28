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

package sqlite

import (
	"fmt"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"
)

func (sqlite *SSqliteBackend) CommitTableChangeSQL(ts sqlchemy.ITableSpec, changes sqlchemy.STableChanges) []string {
	ret := make([]string, 0)

	for _, idx := range changes.RemoveIndexes {
		sql := fmt.Sprintf("DROP INDEX IF EXISTS `%s`.`%s`", ts.Name(), idx.Name())
		ret = append(ret, sql)
		log.Infof("%s;", sql)
	}

	needNewTable := false

	// first check if primary key is modifed
	changePrimary := false
	oldHasPrimary := false
	for _, col := range changes.RemoveColumns {
		if col.IsPrimary() {
			changePrimary = true
			oldHasPrimary = true
		}
	}
	for _, cols := range changes.UpdatedColumns {
		if cols.OldCol.IsPrimary() != cols.NewCol.IsPrimary() {
			changePrimary = true
		}
		if cols.OldCol.IsPrimary() {
			oldHasPrimary = true
		}
	}
	for _, col := range changes.AddColumns {
		if col.IsPrimary() {
			changePrimary = true
		}
	}
	if changePrimary && oldHasPrimary {
		needNewTable = true
	}
	/* IGNORE DROP STATEMENT */
	for _, col := range changes.RemoveColumns {
		sql := fmt.Sprintf("DROP COLUMN `%s`", col.Name())
		log.Debugf("skip ALTER TABLE %s %s;", ts.Name(), sql)
		// alters = append(alters, sql)
		// ignore drop statement
		// if the column is auto_increment integer column,
		// then need to drop auto_increment attribute
		if col.IsAutoIncrement() {
			needNewTable = true
		}
		// if the column is not nullable but no default
		// then need to drop the not-nullable attribute
		if !col.IsNullable() && col.Default() == "" {
			needNewTable = true
		}
	}
	if len(changes.UpdatedColumns) > 0 {
		needNewTable = true
	}
	for _, col := range changes.AddColumns {
		sql := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN %s", ts.Name(), col.DefinitionString())
		ret = append(ret, sql)
	}
	if changePrimary {
		needNewTable = true
	}

	if needNewTable {
		newTableName := fmt.Sprintf("%s_tmp", ts.Name())
		oldTableName := fmt.Sprintf("%s_old", ts.Name())
		// create a table with alter name
		// var newTable *sqlchemy.STableSpec
		newTable := ts.(*sqlchemy.STableSpec).Clone(newTableName, 0)
		createSqls := newTable.CreateSQLs()
		ret = append(ret, createSqls...)
		// insert
		colNameMap := make(map[string]string)

		for _, cols := range changes.UpdatedColumns {
			if cols.OldCol.Name() != cols.NewCol.Name() {
				colNameMap[cols.NewCol.Name()] = cols.OldCol.Name()
			}

		}
		colNames := make([]string, 0)
		srcCols := make([]string, 0)
		for _, col := range ts.Columns() {
			colName := col.Name()
			srcName := colName
			if n, ok := colNameMap[colName]; ok {
				srcName = n
			}
			colNames = append(colNames, fmt.Sprintf("`%s`", colName))
			srcCols = append(srcCols, fmt.Sprintf("`%s`", srcName))
		}
		sql := fmt.Sprintf("INSERT INTO `%s`(%s) SELECT %s FROM `%s`", newTableName, strings.Join(colNames, ", "), strings.Join(srcCols, ", "), ts.Name())
		ret = append(ret, sql)
		// change name
		sql = fmt.Sprintf("ALTER TABLE `%s` RENAME TO `%s`", ts.Name(), oldTableName)
		ret = append(ret, sql)
		sql = fmt.Sprintf("ALTER TABLE `%s` RENAME TO `%s`", newTableName, ts.Name())
		ret = append(ret, sql)
	}

	for _, idx := range changes.AddIndexes {
		sql := createIndexSQL(ts, idx)
		ret = append(ret, sql)
		log.Infof("%s;", sql)
	}

	return ret
}

func createIndexSQL(ts sqlchemy.ITableSpec, idx sqlchemy.STableIndex) string {
	return fmt.Sprintf("CREATE INDEX `%s` ON `%s` (%s)", idx.Name(), ts.Name(), strings.Join(idx.QuotedColumns("`"), ","))
}
