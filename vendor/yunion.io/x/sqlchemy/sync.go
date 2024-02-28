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

func compareColumnIndex(c1, c2 IColumnSpec) int {
	i1 := c1.GetColIndex()
	i2 := c2.GetColIndex()
	return i1 - i2
}

type SUpdateColumnSpec struct {
	OldCol IColumnSpec
	NewCol IColumnSpec
}

func DiffCols(tableName string, cols1 []IColumnSpec, cols2 []IColumnSpec) ([]IColumnSpec, []SUpdateColumnSpec, []IColumnSpec) {
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
	update := make([]SUpdateColumnSpec, 0)
	add := make([]IColumnSpec, 0)
	for i < len(cols1) || j < len(cols2) {
		if i < len(cols1) && j < len(cols2) {
			comp := compareColumnSpec(cols1[i], cols2[j])
			if comp == 0 {
				if cols1[i].DefinitionString() != cols2[j].DefinitionString() || cols1[i].IsPrimary() != cols2[j].IsPrimary() {
					log.Infof("UPDATE %s: %s(primary:%v) => %s(primary:%v)", tableName, cols1[i].DefinitionString(), cols1[i].IsPrimary(), cols2[j].DefinitionString(), cols2[j].IsPrimary())
					update = append(update, SUpdateColumnSpec{
						OldCol: cols1[i],
						NewCol: cols2[j],
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
	for i := 0; i < len(add); {
		intCol := add[i].(iColumnInternal)
		if len(intCol.Oldname()) > 0 {
			// find delete column
			rmIdx := -1
			for j := range remove {
				if remove[j].Name() == intCol.Oldname() {
					// remove from
					rmIdx = j
					break
				}
			}
			if rmIdx >= 0 {
				oldCol := remove[rmIdx]
				{
					// remove from remove
					copy(remove[rmIdx:], remove[rmIdx+1:])
					remove = remove[:len(remove)-1]
				}
				{
					// remove from add
					copy(add[i:], add[i+1:])
					add = add[:len(add)-1]
				}
				{
					update = append(update, SUpdateColumnSpec{
						OldCol: oldCol,
						NewCol: intCol,
					})
				}
				// do not increase i
				continue
			}
		}
		i++
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

	qChar := db.backend.QuoteChar()

	if db.backend.IsSupportIndexAndContraints() {
		_, constraints, err := ts.fetchIndexesAndConstraints()
		if err != nil {
			if errors.Cause(err) != ErrTableNotExists {
				log.Errorf("fetchIndexesAndConstraints fail %s", err)
			}
			return nil
		}

		for _, constraint := range constraints {
			sql := fmt.Sprintf("ALTER TABLE %s%s%s DROP FOREIGN KEY %s%s%s", qChar, ts.name, qChar, qChar, constraint.name, qChar)
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

// Drop drop table
func (ts *STableSpec) Drop() error {
	if !ts.Exists() {
		return nil
	}
	db := ts.Database()
	if db == nil {
		panic("DropForeignKeySQL empty database")
	}
	if db.backend == nil {
		panic("DropForeignKeySQL empty backend")
	}
	sql := db.backend.DropTableSQL(ts.name)
	_, err := db.Exec(sql)
	if err != nil {
		log.Errorf("exec sql error %s: %s", sql, err)
		return errors.Wrap(err, "Exec")
	}
	return nil
}

type STableChanges struct {
	// indexes
	RemoveIndexes []STableIndex
	AddIndexes    []STableIndex

	// Columns
	RemoveColumns  []IColumnSpec
	UpdatedColumns []SUpdateColumnSpec
	AddColumns     []IColumnSpec

	OldColumns []IColumnSpec
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
		addIndexes, removeIndexes = diffIndexes(indexes, ts._indexes)
	}

	cols, err := ts.Database().backend.FetchTableColumnSpecs(ts)
	if err != nil {
		log.Errorf("fetchColumnDefs fail: %s", err)
		return nil
	}

	remove, update, add := DiffCols(ts.name, cols, ts.Columns())

	return ts.Database().backend.CommitTableChangeSQL(ts, STableChanges{
		RemoveIndexes:  removeIndexes,
		AddIndexes:     addIndexes,
		RemoveColumns:  remove,
		UpdatedColumns: update,
		AddColumns:     add,
		OldColumns:     cols,
	})
}

// Sync executes the SQLs to synchronize the DB definion of s SQL database
// by applying the SQL statements generated by SyncSQL()
func (ts *STableSpec) Sync() error {
	sqls := ts.SyncSQL()
	if sqls != nil {
		for _, sql := range sqls {
			log.Infof(sql)
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
