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
	"sort"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
)

func (mysql *SDamengBackend) CommitTableChangeSQL(ts sqlchemy.ITableSpec, changes sqlchemy.STableChanges) []string {
	ret := make([]string, 0)

	for _, idx := range changes.RemoveIndexes {
		sql := fmt.Sprintf(`DROP INDEX "%s"`, idx.Name())
		ret = append(ret, sql)
		log.Infof("%s;", sql)
	}

	alters := make([]string, 0)

	// first check if primary key is modifed
	changePrimary := false
	for _, col := range changes.RemoveColumns {
		if col.IsPrimary() {
			changePrimary = true
			break
		}
	}
	if !changePrimary {
		for _, cols := range changes.UpdatedColumns {
			if cols.OldCol.IsPrimary() != cols.NewCol.IsPrimary() {
				changePrimary = true
				break
			}
		}
	}
	if !changePrimary {
		for _, col := range changes.AddColumns {
			if col.IsPrimary() {
				changePrimary = true
				break
			}
		}
	}
	// in case of a primary key change, we first need to drop primary key.
	// BUT if a mysql table has no primary key at all,
	// exec drop primary key will cause error
	if changePrimary {
		oldHasPrimary := false
		for _, col := range changes.OldColumns {
			if col.IsPrimary() {
				oldHasPrimary = true
				break
			}
		}
		if oldHasPrimary {
			alters = append(alters, fmt.Sprintf(`ALTER TABLE "%s" DROP PRIMARY KEY;`, ts.Name()))
		}
	}

	/* IGNORE DROP STATEMENT */
	for _, col := range changes.RemoveColumns {
		sql := fmt.Sprintf(`ALTER TABLE "%s" DROP COLUMN "%s";`, ts.Name(), col.Name())
		log.Debugf("skip %s", sql)
		// alters = append(alters, sql)
		// ignore drop statement
		// if the column is auto_increment integer column,
		// then need to drop auto_increment attribute
		if col.IsAutoIncrement() {
			// make sure the column is nullable
			col.SetNullable(true)
			log.Errorf("column %s is IDENTITY, drop IDENTITY attribute", col.Name())
			col.SetAutoIncrement(false)
			sql := fmt.Sprintf(`ALTER TABLE "%s" DROP IDENTITY;`, ts.Name())
			alters = append(alters, sql)
			sql = fmt.Sprintf(`ALTER TABLE "%s" MODIFY (%s);`, ts.Name(), col.DefinitionString())
			alters = append(alters, sql)
		}
		// if the column is not nullable but no default
		// then need to drop the not-nullable attribute
		if !col.IsNullable() && col.Default() == "" {
			col.SetNullable(true)
			sql := fmt.Sprintf(`ALTER TABLE "%s" MODIFY (%s);`, ts.Name(), col.DefinitionString())
			alters = append(alters, sql)
			log.Errorf("column %s is not nullable but no default, drop not nullable attribute", col.Name())
		}
	}
	for _, cols := range changes.UpdatedColumns {
		if cols.OldCol.Name() != cols.NewCol.Name() {
			// rename
			sql := fmt.Sprintf(`ALTER TABLE "%s" RENAME COLUMN "%s" TO "%s";`, ts.Name(), cols.OldCol.Name(), cols.NewCol.Name())
			alters = append(alters, sql)
		} else if (strings.HasPrefix(cols.OldCol.ColType(), "VARCHAR") && cols.NewCol.ColType() == "TEXT") || (strings.HasPrefix(cols.NewCol.ColType(), "VARCHAR") && cols.OldCol.ColType() == "TEXT") {
			// change varchar to text
			oldTmpName := fmt.Sprintf("%s_%s", cols.OldCol.Name(), utils.GenRequestId(6))
			alters = append(alters,
				// rename oldcol
				fmt.Sprintf(`ALTER TABLE "%s" RENAME COLUMN "%s" TO "%s";`, ts.Name(), cols.OldCol.Name(), oldTmpName),
				// add a new col
				fmt.Sprintf(`ALTER TABLE "%s" ADD %s;`, ts.Name(), cols.NewCol.DefinitionString()),
				// copy data
				fmt.Sprintf(`UPDATE "%s" SET "%s"=TRIM("%s");`, ts.Name(), cols.NewCol.Name(), oldTmpName),
				// drop old col
				fmt.Sprintf(`ALTER TABLE "%s" DROP COLUMN "%s";`, ts.Name(), oldTmpName),
			)
		} else {
			if cols.OldCol.IsAutoIncrement() {
				sql := fmt.Sprintf(`ALTER TABLE "%s" DROP IDENTITY;`, ts.Name())
				alters = append(alters, sql)
			}
			if cols.NewCol.IsAutoIncrement() {
				sql := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" IDENTITY(%d, 1);`, ts.Name(), cols.NewCol.Name(), cols.NewCol.AutoIncrementOffset())
				alters = append(alters, sql)
			} else {
				sql := fmt.Sprintf(`ALTER TABLE "%s" MODIFY (%s)`, ts.Name(), cols.NewCol.DefinitionString())
				alters = append(alters, sql)
			}

			if hasDefault(cols.NewCol) && cols.OldCol.Default() != cols.NewCol.Default() {
				defStr := cols.NewCol.Default()
				defStr = sqlchemy.GetStringValue(cols.NewCol.ConvertFromString(defStr))
				if cols.NewCol.IsText() {
					defStr = "'" + defStr + "'"
				}
				sql := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" SET DEFAULT %s`, ts.Name(), cols.NewCol.Name(), defStr)
				alters = append(alters, sql)
			} else if !hasDefault(cols.NewCol) && hasDefault(cols.OldCol) {
				sql := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN "%s" DROP DEFAULT`, ts.Name(), cols.NewCol.Name())
				alters = append(alters, sql)
			}
		}
	}
	for _, col := range changes.AddColumns {
		sql := fmt.Sprintf(`ALTER TABLE "%s" ADD %s`, ts.Name(), col.DefinitionString())
		alters = append(alters, sql)
	}
	if changePrimary {
		primaries := make([]string, 0)
		primariesQuote := make([]string, 0)
		for _, c := range ts.Columns() {
			if c.IsPrimary() {
				primaries = append(primaries, c.Name())
				primariesQuote = append(primariesQuote, fmt.Sprintf(`"%s"`, c.Name()))
			}
		}
		if len(primaries) > 0 {
			sort.Strings(primaries)
			pkName := fmt.Sprintf("pk_%s_%s", ts.Name(), strings.Join(primaries, "_"))
			sql := fmt.Sprintf(`ALTER TABLE "%s" ADD CONSTRAINT %s PRIMARY KEY(%s)`, ts.Name(), pkName, strings.Join(primariesQuote, ", "))
			alters = append(alters, sql)
		}
	}

	if len(alters) > 0 {
		ret = append(ret, alters...)
		log.Infof("%s", strings.Join(alters, "\n"))
	}

	for _, idx := range changes.AddIndexes {
		sql := createIndexSQL(ts, idx)
		ret = append(ret, sql)
		log.Infof("%s", sql)
	}

	return ret
}

func createIndexSQL(ts sqlchemy.ITableSpec, idx sqlchemy.STableIndex) string {
	return fmt.Sprintf(`CREATE INDEX "%s" ON "%s" (%s);`, idx.Name(), ts.Name(), strings.Join(idx.QuotedColumns(`"`), ","))
}
