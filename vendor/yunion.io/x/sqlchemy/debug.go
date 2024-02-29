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
	"strings"
	"time"

	"yunion.io/x/log"
)

var (
	// DEBUG_SQLCHEMY is a global constant that indicates turn on SQL debug
	DEBUG_SQLCHEMY = false
)

func sqlDebug(key, sqlstr string, variables []interface{}) {
	sqlstr = _sqlDebug(sqlstr, variables)
	if key == "" {
		key = "SQUery"
	}
	log.Debugln(key, sqlstr)
}

func _sqlDebug(sqlstr string, variables []interface{}) string {
	return SQLPrintf(sqlstr, variables)
}

func SQLPrintf(sqlstr string, variables []interface{}) string {
	for _, v := range variables {
		switch v.(type) {
		case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			sqlstr = strings.Replace(sqlstr, "?", fmt.Sprintf("%v", v), 1)
		case string, time.Time:
			sqlstr = strings.Replace(sqlstr, "?", fmt.Sprintf("'%s'", v), 1)
		default:
			sqlstr = strings.Replace(sqlstr, "?", fmt.Sprintf("'%v'", v), 1)
		}
	}
	return sqlstr
}

// DebugQuery show the full query string for debug
func (tq *SQuery) DebugQuery() {
	tq.DebugQuery2("")
}

// DebugQuery show the full query string for debug
func (tq *SQuery) DebugQuery2(key string) {
	sqlstr := tq.String()
	vars := tq.Variables()
	sqlDebug(key, sqlstr, vars)
}

func (tq *SQuery) DebugString() string {
	return _sqlDebug(tq.String(), tq.Variables())
}

// DebugQuery show the full query string for a subquery for debug
func (sqf *SSubQuery) DebugQuery2(key string) {
	sqlstr := sqf.Expression()
	vars := sqf.query.Variables()
	sqlDebug(key, sqlstr, vars)
}

// DebugQuery show the full query string for a subquery for debug
func (sqf *SSubQuery) DebugQuery() {
	sqf.DebugQuery2("")
}

// DebugInsert does insert with debug mode on
func (t *STableSpec) DebugInsert(dt interface{}) error {
	return t.insert(dt, false, true)
}

// DebugInsertOrUpdate does insertOrUpdate with debug mode on
func (t *STableSpec) DebugInsertOrUpdate(dt interface{}) error {
	return t.insert(dt, true, true)
}

// DebugUpdateFields does update with debug mode on
func (t *STableSpec) DebugUpdateFields(dt interface{}, fields map[string]interface{}) error {
	return t.updateFields(dt, fields, true)
}
