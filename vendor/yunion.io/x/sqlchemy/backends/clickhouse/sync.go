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
	"sort"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/sortedstring"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
)

func findTtlColumn(cols []sqlchemy.IColumnSpec) sColumnTTL {
	ret := sColumnTTL{}
	for _, col := range cols {
		if clickCol, ok := col.(IClickhouseColumnSpec); ok {
			c, u := clickCol.GetTTL()
			if c > 0 && len(u) > 0 {
				ret = sColumnTTL{
					ColName: clickCol.Name(),
					sTTL: sTTL{
						Count: c,
						Unit:  u,
					},
				}
			}
		}
	}
	return ret
}

func findPartitions(cols []sqlchemy.IColumnSpec) []string {
	parts := make([]string, 0)
	for i := range cols {
		if c, ok := cols[i].(IClickhouseColumnSpec); ok {
			part := strings.ReplaceAll(c.PartitionBy(), " ", "")
			if len(part) > 0 && !utils.IsInStringArray(part, parts) {
				parts = append(parts, part)
			}
		}
	}
	sort.Strings(parts)
	return parts
}

func arrayContainsWord(strs []string, word string) bool {
	for _, str := range strs {
		if stringutils.ContainsWord(str, word) {
			return true
		}
	}
	return false
}

func (clickhouse *SClickhouseBackend) CommitTableChangeSQL(ts sqlchemy.ITableSpec, changes sqlchemy.STableChanges) []string {
	ret := make([]string, 0)

	needCopyTable := false

	alters := make([]string, 0)
	// first check if primary key is modifed
	changePrimary := false

	for _, col := range changes.RemoveColumns {
		if col.IsPrimary() {
			changePrimary = true
		}
	}

	for _, cols := range changes.UpdatedColumns {
		if cols.OldCol.IsPrimary() != cols.NewCol.IsPrimary() {
			changePrimary = true
		}
	}
	for _, col := range changes.AddColumns {
		if col.IsPrimary() {
			changePrimary = true
		}
	}

	if changePrimary {
		log.Infof("primary key changed")
		needCopyTable = true
	}
	// if changePrimary && oldHasPrimary {
	// 	sql := fmt.Sprintf("DROP PRIMARY KEY")
	// 	alters = append(alters, sql)
	// }
	/* IGNORE DROP STATEMENT */
	for _, col := range changes.RemoveColumns {
		sql := fmt.Sprintf("DROP COLUMN `%s`", col.Name())
		log.Debugf("skip ALTER TABLE %s %s;", ts.Name(), sql)
		// alters = append(alters, sql)
		// ignore drop statement
		// if the column is auto_increment integer column,
		// then need to drop auto_increment attribute
		if col.IsAutoIncrement() {
			// make sure the column is nullable
			col.SetNullable(true)
			log.Errorf("column %s is auto_increment, drop auto_inrement attribute", col.Name())
			col.SetAutoIncrement(false)
			sql := fmt.Sprintf("MODIFY COLUMN %s", col.DefinitionString())
			alters = append(alters, sql)
		}
		// if the column is not nullable but no default
		// then need to drop the not-nullable attribute
		if !col.IsNullable() && col.Default() == "" {
			col.SetNullable(true)
			sql := fmt.Sprintf("MODIFY COLUMN %s", col.DefinitionString())
			alters = append(alters, sql)
			log.Errorf("column %s is not nullable but no default, drop not nullable attribute", col.Name())
		}
	}

	oldPartitions := findPartitions(changes.OldColumns)
	for _, cols := range changes.UpdatedColumns {
		if cols.OldCol.IsNullable() && !cols.NewCol.IsNullable() && arrayContainsWord(oldPartitions, cols.NewCol.Name()) {
			needCopyTable = true
		} else {
			sql := fmt.Sprintf("MODIFY COLUMN %s", cols.NewCol.DefinitionString())
			alters = append(alters, sql)
		}
	}
	for _, col := range changes.AddColumns {
		sql := fmt.Sprintf("ADD COLUMN %s", col.DefinitionString())
		alters = append(alters, sql)
	}
	/*if changePrimary {
		primaries := make([]string, 0)
		for _, c := range ts.Columns() {
			if c.IsPrimary() {
				primaries = append(primaries, fmt.Sprintf("`%s`", c.Name()))
			}
		}
		if len(primaries) > 0 {
			sql := fmt.Sprintf("ADD PRIMARY KEY(%s)", strings.Join(primaries, ", "))
			alters = append(alters, sql)
		}
	}*/

	// check TTL
	oldTtlSpec := findTtlColumn(changes.OldColumns)
	newTtlSpec := findTtlColumn(ts.Columns())
	log.Debugf("old: %s new: %s", jsonutils.Marshal(oldTtlSpec), jsonutils.Marshal(newTtlSpec))
	if oldTtlSpec != newTtlSpec {
		if oldTtlSpec.Count > 0 && newTtlSpec.Count == 0 {
			// remove
			sql := fmt.Sprintf("REMOVE TTL")
			alters = append(alters, sql)
		} else {
			// alter
			sql := fmt.Sprintf("MODIFY TTL `%s` + INTERVAL %d %s", newTtlSpec.ColName, newTtlSpec.Count, newTtlSpec.Unit)
			alters = append(alters, sql)
		}
	}

	if len(alters) > 0 {
		sql := fmt.Sprintf("ALTER TABLE `%s` %s;", ts.Name(), strings.Join(alters, ", "))
		ret = append(ret, sql)
	}

	// check partitions
	newPartitions := findPartitions(ts.Columns())
	if !sortedstring.Equals(oldPartitions, newPartitions) {
		log.Infof("partition inconsistemt: old=%s new=%s", oldPartitions, newPartitions)
		needCopyTable = true
	}

	// needCopyTable
	if needCopyTable {
		// create new table
		alterTableName := fmt.Sprintf("%s_tmp_%d", ts.Name(), time.Now().Unix())
		alterTable := ts.(*sqlchemy.STableSpec).Clone(alterTableName, 0)
		createSqls := alterTable.CreateSQLs()
		ret = append(ret, createSqls...)
		colNames := make([]string, 0)
		for _, c := range ts.Columns() {
			colNames = append(colNames, fmt.Sprintf("`%s`", c.Name()))
		}
		colNamesStr := strings.Join(colNames, ",")
		// copy data
		sql := fmt.Sprintf("INSERT INTO `%s` (%s) SELECT %s FROM `%s`", alterTableName, colNamesStr, colNamesStr, ts.Name())
		ret = append(ret, sql)
		// rename tables
		sql = fmt.Sprintf("RENAME TABLE `%s` TO `%s_backup`", ts.Name(), alterTableName)
		ret = append(ret, sql)
		sql = fmt.Sprintf("RENAME TABLE `%s` TO `%s`", alterTableName, ts.Name())
		ret = append(ret, sql)
	}

	return ret
}
