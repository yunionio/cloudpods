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

import (
	"fmt"
	"sort"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

func (ts *STableSpec) fetchIndexesAndConstraints() ([]STableIndex, []STableConstraint, error) {
	return ts.Database().backend.FetchIndexesAndConstraints(ts)
}

func compareColumnSpec(c1, c2 IColumnSpec) int {
	return strings.Compare(c1.Name(), c2.Name())
}

type sUpdateColumnSpec struct {
	oldCol IColumnSpec
	newCol IColumnSpec
}

func diffCols(tableName string, cols1 []IColumnSpec, cols2 []IColumnSpec) ([]IColumnSpec, []sUpdateColumnSpec, []IColumnSpec) {
	sort.Slice(cols1, func(i, j int) bool {
		return compareColumnSpec(cols1[i], cols1[j]) < 0
	})
	sort.Slice(cols2, func(i, j int) bool {
		return compareColumnSpec(cols2[i], cols2[j]) < 0
	})
	// for i := range cols1 {
	// 	log.Debugf("%s %v", cols1[i].DefinitionString(), cols1[i].IsPrimary())
	// }
	// for i := range cols2 {
	// 	log.Debugf("%s %v", cols2[i].DefinitionString(), cols2[i].IsPrimary())
	// }
	i := 0
	j := 0
	remove := make([]IColumnSpec, 0)
	update := make([]sUpdateColumnSpec, 0)
	add := make([]IColumnSpec, 0)
	for i < len(cols1) || j < len(cols2) {
		if i < len(cols1) && j < len(cols2) {
			comp := compareColumnSpec(cols1[i], cols2[j])
			if comp == 0 {
				if cols1[i].DefinitionString() != cols2[j].DefinitionString() || cols1[i].IsPrimary() != cols2[j].IsPrimary() {
					log.Infof("UPDATE %s: %s(primary:%v) => %s(primary:%v)", tableName, cols1[i].DefinitionString(), cols1[i].IsPrimary(), cols2[j].DefinitionString(), cols2[j].IsPrimary())
					update = append(update, sUpdateColumnSpec{
						oldCol: cols1[i],
						newCol: cols2[j],
					})
				}
				i++
				j++
			} else if comp > 0 {
				add = append(add, cols2[j])
				j++
			} else {
				remove = append(remove, cols1[i])
				i++
			}
		} else if i < len(cols1) {
			remove = append(remove, cols1[i])
			i++
		} else if j < len(cols2) {
			add = append(add, cols2[j])
			j++
		}
	}
	return remove, update, add
}

func diffIndexes2(exists []STableIndex, defs []STableIndex) (diff []STableIndex) {
	diff = make([]STableIndex, 0)
	for i := 0; i < len(exists); i++ {
		findDef := false
		for j := 0; j < len(defs); j++ {
			if defs[j].IsIdentical(exists[i].columns...) {
				findDef = true
				break
			}
		}
		if !findDef {
			diff = append(diff, exists[i])
		}
	}
	return
}

func diffIndexes(exists []STableIndex, defs []STableIndex) (added []STableIndex, removed []STableIndex) {
	return diffIndexes2(defs, exists), diffIndexes2(exists, defs)
}

// DropForeignKeySQL returns the SQL statements to do droping foreignkey for a TableSpec
func (ts *STableSpec) DropForeignKeySQL() []string {
	ret := make([]string, 0)

	db := ts.Database()
	if db == nil {
		panic("DropForeignKeySQL empty database")
	}
	if db.backend == nil {
		panic("DropForeignKeySQL empty backend")
	}
	if db.backend.IsSupportIndexAndContraints() {
		_, constraints, err := ts.fetchIndexesAndConstraints()
		if err != nil {
			if errors.Cause(err) != ErrTableNotExists {
				log.Errorf("fetchIndexesAndConstraints fail %s", err)
			}
			return nil
		}

		for _, constraint := range constraints {
			sql := fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `%s`", ts.name, constraint.name)
			ret = append(ret, sql)
			log.Infof("%s;", sql)
		}
	}

	return ret
}

// Exists checks wheter a table exists
func (ts *STableSpec) Exists() bool {
	tables := ts.Database().GetTables()
	in, _ := utils.InStringArray(ts.name, tables)
	return in
}

// SyncSQL returns SQL statements that make table in database consistent with TableSpec definitions
// by comparing table definition derived from TableSpec and that in database
func (ts *STableSpec) SyncSQL() []string {
	if !ts.Exists() {
		log.Debugf("table %s not created yet", ts.name)
		return ts.CreateSQLs()
	}

	var addIndexes, removeIndexes []STableIndex

	if ts.Database().backend.IsSupportIndexAndContraints() {
		indexes, _, err := ts.fetchIndexesAndConstraints()
		if err != nil {
			if errors.Cause(err) != ErrTableNotExists {
				log.Errorf("fetchIndexesAndConstraints fail %s", err)
			}
			return nil
		}
		addIndexes, removeIndexes = diffIndexes(indexes, ts.indexes)
	}

	ret := make([]string, 0)
	cols, err := ts.Database().backend.FetchTableColumnSpecs(ts)
	if err != nil {
		log.Errorf("fetchColumnDefs fail: %s", err)
		return nil
	}

	for _, idx := range removeIndexes {
		sql := templateEval(ts.Database().backend.DropIndexSQLTemplate(), struct {
			Table string
			Index string
		}{
			Table: ts.name,
			Index: idx.name,
		})
		ret = append(ret, sql)
		log.Infof("%s;", sql)
	}

	alters := make([]string, 0)
	remove, update, add := diffCols(ts.name, cols, ts.columns)
	// first check if primary key is modifed
	changePrimary := false
	for _, col := range remove {
		if col.IsPrimary() {
			changePrimary = true
		}
	}
	for _, cols := range update {
		if cols.oldCol.IsPrimary() != cols.newCol.IsPrimary() {
			changePrimary = true
		}
	}
	for _, col := range add {
		if col.IsPrimary() {
			changePrimary = true
		}
	}
	if changePrimary {
		oldHasPrimary := false
		for _, c := range cols {
			if c.IsPrimary() {
				oldHasPrimary = true
				break
			}
		}
		if oldHasPrimary {
			sql := fmt.Sprintf("DROP PRIMARY KEY")
			alters = append(alters, sql)
		}
	}
	/* IGNORE DROP STATEMENT */
	for _, col := range remove {
		sql := fmt.Sprintf("DROP COLUMN `%s`", col.Name())
		log.Infof("ALTER TABLE %s %s;", ts.name, sql)
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
	for _, cols := range update {
		sql := fmt.Sprintf("MODIFY COLUMN %s", cols.newCol.DefinitionString())
		alters = append(alters, sql)
	}
	for _, col := range add {
		sql := fmt.Sprintf("ADD COLUMN %s", col.DefinitionString())
		alters = append(alters, sql)
	}
	if changePrimary {
		primaries := make([]string, 0)
		for _, c := range ts.columns {
			if c.IsPrimary() {
				primaries = append(primaries, fmt.Sprintf("`%s`", c.Name()))
			}
		}
		if len(primaries) > 0 {
			sql := fmt.Sprintf("ADD PRIMARY KEY(%s)", strings.Join(primaries, ", "))
			alters = append(alters, sql)
		}
	}

	if len(alters) > 0 {
		sql := fmt.Sprintf("ALTER TABLE `%s` %s;", ts.name, strings.Join(alters, ", "))
		ret = append(ret, sql)
	}

	for _, idx := range addIndexes {
		sql := fmt.Sprintf("CREATE INDEX `%s` ON `%s` (%s)", idx.name, ts.name, strings.Join(idx.QuotedColumns(), ","))
		ret = append(ret, sql)
		log.Infof("%s;", sql)
	}

	return ret
}

// Sync executes the SQLs to synchronize the DB definion of s SQL database
// by applying the SQL statements generated by SyncSQL()
func (ts *STableSpec) Sync() error {
	sqls := ts.SyncSQL()
	if sqls != nil {
		for _, sql := range sqls {
			_, err := ts.Database().Exec(sql)
			if err != nil {
				log.Errorf("exec sql error %s: %s", sql, err)
				return err
			}
		}
	}
	return nil
}

// CheckSync checks whether the table in database consistent with TableSpec
func (ts *STableSpec) CheckSync() error {
	sqls := ts.SyncSQL()
	if len(sqls) > 0 {
		for _, sql := range sqls {
			fmt.Println(sql)
		}
		return fmt.Errorf("DB table %q not in sync", ts.name)
	}
	return nil
}
