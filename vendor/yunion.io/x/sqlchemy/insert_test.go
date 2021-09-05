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
	"reflect"
	"testing"

	"yunion.io/x/pkg/util/reflectutils"
)

func insertSqlPrep(v interface{}) (string, []interface{}, error) {
	vvvalue := reflect.ValueOf(v).Elem()
	vv := vvvalue.Interface()
	vvFields := reflectutils.FetchStructFieldValueSet(vvvalue)
	SetDefaultDB(nil)
	ts := NewTableSpecFromStruct(vv, "vv")
	sql, vals, err := ts.insertSqlPrep(vvFields, false)
	return sql, vals, err
}

func TestInsertAutoIncrement(t *testing.T) {
	sql, vals, err := insertSqlPrep(&struct {
		RowId int `auto_increment:"true"`
	}{})
	if err != nil {
		t.Errorf("prepare sql failed: %s", err)
		return
	}
	wantSql := "INSERT INTO `vv` () VALUES()"
	if sql != wantSql {
		t.Errorf("sql want: %s\ngot: %s", wantSql, sql)
		return
	}
	if len(vals) != 0 {
		t.Errorf("vals want %d, got %d", 0, len(vals))
		return
	}
}

func TestInsertMultiAutoIncrement(t *testing.T) {
	defer func() {
		v := recover()
		if v == nil {
			t.Errorf("should panic with multiple auto_increment fields")
		}
	}()
	_, _, err := insertSqlPrep(&struct {
		RowId  int `auto_increment:"true"`
		RowId2 int `auto_increment:"true"`
	}{})
	t.Errorf("should panic but it continues: err: %s", err)
}

func TestInsertWithPointerValue(t *testing.T) {
	sql, vals, err := insertSqlPrep(&struct {
		RowId int `auto_increment:"true"`
		ColT1 *int
		ColT2 int
		ColT3 string
		ColT4 *string
	}{})
	if err != nil {
		t.Errorf("prepare sql failed: %s", err)
		return
	}
	t.Logf("%s values: %v", sql, vals)
}
